package stdconv

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"log"
	"os"
)

//gocognit:ignore
func TarToZip(
	ctx context.Context,
	tarfile io.Reader,
	zipfile io.Writer,
	verbose bool,
	method uint16,
	maxItemSize int64,
	t2z ConvertHeader,
) error {
	var trdr *tar.Reader = tar.NewReader(tarfile)
	var zwtr *zip.Writer = zip.NewWriter(zipfile)

	ferr := func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			thdr, err := trdr.Next()
			if io.EOF == err { // io.EOF is not wrapped
				return nil
			}
			if nil != err {
				return err
			}

			if tar.TypeReg != thdr.Typeflag {
				if verbose {
					log.Printf(
						"skipping non-regular item(%d): %s\n",
						thdr.Typeflag,
						thdr.Name,
					)
				}
				continue
			}

			var zhdr zip.FileHeader
			err = t2z(thdr, &zhdr)
			if nil != err {
				return err
			}
			zhdr.Method = method

			wtr, err := zwtr.CreateHeader(&zhdr)
			if nil != err {
				return err
			}

			if maxItemSize < thdr.Size {
				if verbose {
					log.Printf("too big file(%v): %s\n", thdr.Size, thdr.Name)
					log.Printf(
						"make ENV_MAX_ITEM_SIZE larger to keep the original content\n",
					)
				}
			}

			limited := &io.LimitedReader{
				R: trdr,
				N: maxItemSize,
			}

			_, err = io.Copy(wtr, limited)
			if nil != err {
				return err
			}
		}
	}()

	return errors.Join(ferr, zwtr.Close())
}

type ConvertHeader func(*tar.Header, *zip.FileHeader) error

func LeastConvertHeader(input *tar.Header, output *zip.FileHeader) error {
	var finfo fs.FileInfo = input.FileInfo()
	zhdr, e := zip.FileInfoHeader(finfo)
	if nil != e {
		return e
	}

	zhdr.Name = input.Name
	*output = *zhdr

	return nil
}

type ConvertConfig struct {
	MaxItemSize int64
	ConvertHeader
	Method  uint16
	Verbose bool
}

const MaxItemSizeDefault int64 = 16777216

var ConvertConfigDefault ConvertConfig = ConvertConfig{
	MaxItemSize:   MaxItemSizeDefault,
	ConvertHeader: LeastConvertHeader,
	Method:        zip.Deflate,
	Verbose:       true,
}

func (c ConvertConfig) WithMaxItemSize(sz int64) ConvertConfig {
	c.MaxItemSize = sz
	return c
}

func (c ConvertConfig) WithConvertHeader(conv ConvertHeader) ConvertConfig {
	c.ConvertHeader = conv
	return c
}

func (c ConvertConfig) WithMethodStore() ConvertConfig {
	c.Method = zip.Store
	return c
}

func (c ConvertConfig) WithMethodDeflate() ConvertConfig {
	c.Method = zip.Deflate
	return c
}

func (c ConvertConfig) WithVerbose(verbose bool) ConvertConfig {
	c.Verbose = verbose
	return c
}

func (c ConvertConfig) WithCompression(compression bool) ConvertConfig {
	switch compression {
	case true:
		return c.WithMethodDeflate()
	default:
		return c.WithMethodStore()
	}
}

func (c ConvertConfig) ConvertToZip(
	ctx context.Context,
	tarfile io.Reader,
	zipfile io.Writer,
) error {
	return TarToZip(
		ctx,
		tarfile,
		zipfile,
		c.Verbose,
		c.Method,
		c.MaxItemSize,
		c.ConvertHeader,
	)
}

func (c ConvertConfig) StdinToTarToZipToStdout(
	ctx context.Context,
) error {
	var rdr io.Reader = bufio.NewReader(os.Stdin)
	var bw *bufio.Writer = bufio.NewWriter(os.Stdout)
	defer bw.Flush()

	return c.ConvertToZip(
		ctx,
		rdr,
		bw,
	)
}
