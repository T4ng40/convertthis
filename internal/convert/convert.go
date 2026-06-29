package convert

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type Format string

const (
	FormatWAV Format = "wav"
	FormatMP3 Format = "mp3"
)

type spec struct {
	Ext        string
	MIME       string
	encodeArgs []string
}

var registry = map[Format]spec{
	FormatWAV: {
		Ext:        "wav",
		MIME:       "audio/wav",
		encodeArgs: []string{"-c:a", "pcm_s16le"},
	},
	FormatMP3: {
		Ext:        "mp3",
		MIME:       "audio/mpeg",
		encodeArgs: []string{"-c:a", "libmp3lame", "-b:a", "192k"},
	},
}

func ParseFormat(s string) (Format, error) {
	f := Format(strings.ToLower(strings.TrimSpace(s)))
	if _, ok := registry[f]; !ok {
		return "", fmt.Errorf("unsupported format %q", s)
	}
	return f, nil
}

func (f Format) Ext() string { return registry[f].Ext }

func (f Format) MIME() string { return registry[f].MIME }

type Converter interface {
	Convert(ctx context.Context, in io.Reader, from, to Format) (io.Reader, error)
}
