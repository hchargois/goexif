package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/hchargois/goexif/exif"
	"github.com/hchargois/goexif/mknote"
	"github.com/hchargois/goexif/tiff"
)

var mnote = flag.Bool("mknote", false, "try to parse makernote data")
var thumb = flag.Bool("thumb", false, "dump thumbail data to stdout (for first listed image file)")

func main() {
	flag.Parse()
	fnames := flag.Args()

	if *mnote {
		exif.RegisterParsers(mknote.All...)
	}

	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	for _, name := range fnames {
		f, err := os.Open(name)
		if err != nil {
			log.Printf("err on %v: %v", name, err)
			continue
		}

		x, err := exif.Decode(f)
		if err != nil {
			log.Printf("err on %v: %v", name, err)
			continue
		}

		if *thumb {
			data, err := x.JpegThumbnail()
			if err != nil {
				log.Fatal("no thumbnail present")
			}
			if _, err := out.Write(data); err != nil {
				log.Fatal(err)
			}
			return
		}

		if len(fnames) > 1 {
			fmt.Fprintf(out, "---- Image '%v' ----\n", name)
		}

		tags := x.Tags()
		nts := make([]namedTag, 0, len(tags))
		longestName := 0
		for name, tag := range tags {
			if len(name) > longestName {
				longestName = len(name)
			}
			nts = append(nts, namedTag{name, tag})
		}
		slices.SortFunc(nts, func(a, b namedTag) int {
			return strings.Compare(string(a.name), string(b.name))
		})

		for _, nt := range nts {
			data, _ := nt.tag.MarshalJSON()
			fmt.Fprintf(out, "    %-*s : %v\n", longestName, nt.name, string(data))
		}
	}
}

type namedTag struct {
	name exif.FieldName
	tag  *tiff.Tag
}
