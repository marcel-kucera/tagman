package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"tagman/patternParser"

	"github.com/bogem/id3v2/v2"
)

type SongMetadata struct {
	Title     string
	Album     string
	Artist    string
	Track     string
	CoverPath string
}

func (m *SongMetadata) merge(def SongMetadata) {
	if m.Title == "" {
		m.Title = def.Title
	}
	if m.Album == "" {
		m.Album = def.Album
	}
	if m.Artist == "" {
		m.Artist = def.Artist
	}
	if m.Track == "" {
		m.Track = def.Track
	}
}

func newMetadataFromMap(m map[string]string) SongMetadata {
	return SongMetadata{
		Title:  m["title"],
		Album:  m["album"],
		Artist: m["artist"],
		Track:  m["track"],
	}
}

func parseOptions(args []string) (SongMetadata, []string) {

	defaultMetadata := SongMetadata{}

	for strings.HasPrefix(args[0], "--") {
		// parse and validate option
		options := strings.SplitN(args[0], "=", 2)
		if len(options) != 2 {
			log.Panicln("invalid option:", args[0])
		}

		// switch over available options
		switch options[0] {
		case "--title":
			defaultMetadata.Title = options[1]
			break
		case "--album":
			defaultMetadata.Album = options[1]
			break
		case "--artist":
			defaultMetadata.Artist = options[1]
			break
		case "--track":
			defaultMetadata.Track = options[1]
			break
		case "--cover":
			defaultMetadata.CoverPath = options[1]
			break
		}

		// goto next option or exit option parsing if no args left (not enough args. will error in the next step)
		if len(args) > 1 {
			args = args[1:]
		} else {
			break
		}
	}

	return defaultMetadata, args
}

var helpText = `tagman [OPTIONS]... PATTERN FILES...

Options:
  Geben den Standardwert an, falls dieser nicht im PATTERN vorkommt
  sollte im format --attribut=wert gegeben werden
  --title Titel
  --album Album
  --artist Künster
  --track Tracknummer
  --cover Dateipfad zum Cover

Pattern:
  Das Pattern besteht aus festen Teilen, die im Dateinahmen vorhanden sein müssen und Tags, welche
  gelesen werden und als Metadaten gesetzt werden

  Beispiel: "%(track). %(title) - %(artist)"`

func main() {
	args := os.Args

	// called without args -> print help
	if len(args) <= 1 {
		fmt.Println(helpText)
		return
	}
	args = args[1:]

	defaultMetadata, args := parseOptions(args)

	// check if there is the pattern and at least one file remaining in args
	if len(args) < 2 {
		// TODO: add better help text
		fmt.Println("Missing pattern or file(s)")
		fmt.Println(helpText)
		return
	}

	// get album cover
	var cover id3v2.PictureFrame
	if defaultMetadata.CoverPath != "" {
		coverData, err := os.ReadFile(defaultMetadata.CoverPath)
		if err != nil {
			log.Panicf("unable to open cover image: %v", err)
		}
		cover = id3v2.PictureFrame{
			Encoding:    id3v2.EncodingUTF8,
			MimeType:    "image/jpeg",
			PictureType: id3v2.PTFrontCover,
			Description: "Front Cover",
			Picture:     coverData,
		}
	}

	// parse pattern
	patternStr := args[0]
	args = args[1:]
	pattern, err := patternParser.Parse(patternStr)
	if err != nil {
		panic(err)
	}

	// at this point only files remain in the args array.
	// we iterate through them, parse filename with the pattern and apply tags
	for _, e := range args {
		assignments, err := pattern.Parse(e)
		if err != nil {
			panic(err)
		}
		metadata := newMetadataFromMap(assignments)
		metadata.merge(defaultMetadata)
		fmt.Printf("%+v\n", metadata)

		tag, err := id3v2.Open(e, id3v2.Options{Parse: true})
		if err != nil {
			fmt.Printf("error opening file %s: %v", e, err)
		}

		tag.SetTitle(metadata.Title)
		tag.SetAlbum(metadata.Album)
		tag.SetArtist(metadata.Artist)
		tag.AddTextFrame(tag.CommonID("Track number/Position in set"), tag.DefaultEncoding(), metadata.Track)
		if defaultMetadata.CoverPath != "" {
			tag.AddAttachedPicture(cover)
		}

		err = tag.Save()
		if err != nil {
			fmt.Printf("error saving file %s: %v", e, err)
		}

	}

}
