package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	cs "github.com/takanoriyanagitani/go-tar2zip/std"
	. "github.com/takanoriyanagitani/go-tar2zip/util"
)

var envValByKey func(string) IO[string] = Lift(
	func(key string) (string, error) {
		val, found := os.LookupEnv(key)
		switch found {
		case true:
			return val, nil
		default:
			return "", fmt.Errorf("env var %s missing", key)
		}
	},
)

var maxItemSizeInt IO[int] = Bind(
	envValByKey("ENV_MAX_ITEM_SIZE"),
	Lift(strconv.Atoi),
).Or(Of(int(cs.MaxItemSizeDefault)))

var useCompression IO[bool] = Bind(
	envValByKey("ENV_USE_DEFLATE"),
	Lift(strconv.ParseBool),
).Or(Of(true))

var verbose IO[bool] = Bind(
	envValByKey("ENV_VERBOSE"),
	Lift(strconv.ParseBool),
).Or(Of(true))

var convertConfig IO[cs.ConvertConfig] = Bind(
	All(
		maxItemSizeInt.ToAny(),
		useCompression.ToAny(),
		verbose.ToAny(),
	),
	Lift(func(a []any) (cs.ConvertConfig, error) {
		return cs.
			ConvertConfigDefault.
			WithMaxItemSize(int64(a[0].(int))).
			WithCompression(a[1].(bool)).
			WithVerbose(a[2].(bool)), nil
	}),
)

var stdin2tar2zip2stdout IO[Void] = Bind(
	convertConfig,
	func(cfg cs.ConvertConfig) IO[Void] {
		return func(ctx context.Context) (Void, error) {
			return Empty, cfg.StdinToTarToZipToStdout(ctx)
		}
	},
)

var sub IO[Void] = func(ctx context.Context) (Void, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	return stdin2tar2zip2stdout(ctx)
}

func main() {
	_, e := sub(context.Background())
	if nil != e {
		log.Printf("%v\n", e)
	}
}
