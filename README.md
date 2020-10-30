# ventil
Valve Key-Value file parser in Go

## installation
`go get -u github.com/noxer/ventil`

## documentation
[https://pkg.go.dev/github.com/noxer/ventil](https://pkg.go.dev/github.com/noxer/ventil)

## usage
The package offers the two functions `ventil.Parse` and `ventil.ParseFile`. `ventil.Parse` takes a reader and parses it into a `*ventil.KV`. `ventil.ParseFile` is a simple wrapper which opens the file and passes it to `ventil.Parse`.

### KV
`*ventil.KV` represents a Key-Value pair. It may contain a value or children, depending on the type of the KV. You can check it with the `HasValue` flag. Be aware that `kv.HasValue == false` does not guarantee, that `kv.FirstChild != nil`, in that case the list of children was empty.

You can call `kv.WriteTo(w)` to write an equivalent representation to the original file but without the comments to w.
