package convert

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
)

type FFmpeg struct {
	Bin string
}

func NewFFmpeg() *FFmpeg { return &FFmpeg{Bin: "ffmpeg"} }

func (f *FFmpeg) bin() string {
	if f.Bin == "" {
		return "ffmpeg"
	}
	return f.Bin
}

func (f *FFmpeg) Available() bool {
	return exec.Command(f.bin(), "-version").Run() == nil
}

func (f *FFmpeg) Convert(ctx context.Context, in io.Reader, from, to Format) (io.Reader, error) {
	// Temp files rather than stdin/stdout pipes: some formats need a seekable
	// output to finalise their header, which a pipe can't provide.
	inFile, err := os.CreateTemp("", "convertthis-in-*."+from.Ext())
	if err != nil {
		return nil, fmt.Errorf("create temp input: %w", err)
	}
	defer os.Remove(inFile.Name())

	if _, err := io.Copy(inFile, in); err != nil {
		inFile.Close()
		return nil, fmt.Errorf("write temp input: %w", err)
	}
	inFile.Close()

	outPath := inFile.Name() + ".out." + to.Ext()
	defer os.Remove(outPath)

	args := []string{"-y", "-i", inFile.Name()}
	args = append(args, registry[to].encodeArgs...)
	args = append(args, outPath)

	cmd := exec.CommandContext(ctx, f.bin(), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("read ffmpeg output: %w", err)
	}
	return bytes.NewReader(data), nil
}
