package ventil

import (
	"fmt"
	"io"
	"strings"
)

// KV represents a Key-Value pair in a Valve Key-Value file.
type KV struct {
	NextSibling *KV
	FirstChild  *KV
	Key         string
	Value       string
	HasValue    bool
}

// Find the first KV identified by key.
func (kv *KV) Find(key string) *KV {
	kv = kv.FirstChild

	for kv != nil {
		if kv.Key == key {
			return kv
		}

		if sub := kv.Find(key); sub != nil {
			return sub
		}

		kv = kv.NextSibling
	}

	return nil
}

// Item returns a subkey by name.
func (kv *KV) Item(key string) *KV {
	kv = kv.FirstChild

	for kv != nil {
		if kv.Key == key {
			return kv
		}

		kv = kv.NextSibling
	}

	return nil
}

// ForEach iterates over the subkeys of a KV and calls f for each KV.
func (kv *KV) ForEach(f func(key string, kv *KV)) {
	kv = kv.FirstChild

	for kv != nil {
		f(kv.Key, kv)
		kv = kv.NextSibling
	}
}

// Tree calls f for each KV in the KV tree, including the KV it was called on.
func (kv *KV) Tree(f func(key string, kv *KV)) {
	for kv != nil {
		f(kv.Key, kv)
		kv.FirstChild.Tree(f)

		kv = kv.NextSibling
	}
}

func (kv *KV) String() string {
	buf := &strings.Builder{}
	printKV(kv, "", buf)
	return buf.String()
}

// WriteTo writes a representation of the KV to w.
func (kv *KV) WriteTo(w io.Writer) (n int64, err error) {
	return printKV(kv, "", w)
}

func printKV(kv *KV, prefix string, w io.Writer) (int64, error) {
	sum := int64(0)
	for kv != nil {
		n, err := fmt.Fprintf(w, "%s\"%s\"", prefix, kv.Key)
		sum += int64(n)
		if err != nil {
			return sum, err
		}

		if !kv.HasValue {
			n, err = fmt.Fprintf(w, "\n%s{\n", prefix)
			sum += int64(n)
			if err != nil {
				return sum, err
			}
			n64, err := printKV(kv.FirstChild, prefix+"\t", w)
			sum += n64
			if err != nil {
				return sum, err
			}
			n, err = fmt.Fprintf(w, "%s}\n", prefix)
			sum += int64(n)
			if err != nil {
				return sum, err
			}
		} else {
			n, err = fmt.Fprintf(w, " \"%s\"\n", kv.Value)
			sum += int64(n)
			if err != nil {
				return sum, err
			}
		}

		kv = kv.NextSibling
	}

	return sum, nil
}
