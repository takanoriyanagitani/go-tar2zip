#!/bin/sh

input=./sample.d/input.tar

output=./sample.d/output.zip

conv="conv=fsync"

geninput(){
	echo generating input...

	mkdir -p sample.d

	echo hw1 > ./sample.d/hw1.txt
	echo hw2 > ./sample.d/hw2.txt

	words=/usr/share/dict/words
	test -f "${words}" && cat "${words}" > ./sample.d/words.txt

	find \
		sample.d \
		-type f \
		-name '*.txt' |
		tar \
			--verbose \
			--create \
			--file "${input}" \
			--files-from=/dev/stdin
}

test -f "${input}" || geninput

export ENV_MAX_ITEM_SIZE=1024
export ENV_MAX_ITEM_SIZE=16777216
export ENV_USE_DEFLATE=true
export ENV_VERBOSE=true

cat "${input}" |
	./tar2zip |
	dd \
		if=/dev/stdin \
		of="${output}" \
		bs=1048576 \
		status=progress \
		$conv

unzip -lv "${output}"

ls -lSh "${input}" "${output}"
