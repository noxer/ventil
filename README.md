# ventil
Valve Key-Value file parser in Go

## installation
`go get -u github.com/noxer/ventil`

## documentation
This Go package allows the parsing of key-value files used in Steam and other Valve products. It maintains the tree-like structure of the files and even supports the `#import` and `#base` directives. Because Valve's KV files are only loosely defined, this package can make no guarantees about compatibility. If you encounter a file that can't be read or is parsed incorrectly, please open an issue. The package name is the German translation of the word "valve".

You can find the API documentation here:
[https://pkg.go.dev/github.com/noxer/ventil](https://pkg.go.dev/github.com/noxer/ventil)

## usage
The package offers the two functions `ventil.Parse` and `ventil.ParseFile`. `ventil.Parse` takes a reader and parses it into a `*ventil.KV`. `ventil.ParseFile` is a simple wrapper which opens the file and passes it to `ventil.Parse`.

### KV
`*ventil.KV` represents a Key-Value pair. It may contain a value or child pairs, depending on the type of the KV. You can check it with the `HasValue` flag. Be aware that `kv.HasValue == false` does not guarantee, that `kv.FirstChild != nil`, in that case the list of children was empty.

You can call `kv.WriteTo(w)` to write an equivalent representation to the original file but without the comments to w.

## features
* Quoted and unquoted keys and values
* Comments
* Escape codes in strings
* Newlines in keys and values
* Subkeys
* Includes
