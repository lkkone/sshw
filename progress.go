package sshw

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	progressBarWidth        = 24
	progressRefreshInterval = 100 * time.Millisecond
)

type transferProgressReader struct {
	reader      io.Reader
	output      io.Writer
	label       string
	total       int64
	transferred int64
	startedAt   time.Time
	lastDraw    time.Time
	enabled     bool
}

type directoryTransferProgress struct {
	output         io.Writer
	label          string
	totalBytes     int64
	totalFiles     int
	transferred    int64
	completedFiles int
	startedAt      time.Time
	lastDraw       time.Time
	enabled        bool
}

type directoryProgressReader struct {
	reader   io.Reader
	progress *directoryTransferProgress
}

func newTransferProgressReader(
	reader io.Reader,
	output io.Writer,
	label string,
	total int64,
) *transferProgressReader {
	return newTransferProgressReaderAt(reader, output, label, total, 0)
}

func newTransferProgressReaderAt(
	reader io.Reader,
	output io.Writer,
	label string,
	total, transferred int64,
) *transferProgressReader {
	return &transferProgressReader{
		reader:      reader,
		output:      output,
		label:       label,
		total:       total,
		transferred: transferred,
		startedAt:   time.Now(),
		enabled:     isTerminalWriter(output),
	}
}

func (p *transferProgressReader) Read(buffer []byte) (int, error) {
	n, err := p.reader.Read(buffer)
	p.transferred += int64(n)
	if n > 0 {
		p.draw(false)
	}
	return n, err
}

func (p *transferProgressReader) Finish() {
	if !p.enabled {
		return
	}
	p.draw(true)
	fmt.Fprintln(p.output)
}

func (p *transferProgressReader) draw(force bool) {
	if !p.enabled {
		return
	}
	now := time.Now()
	if !force && now.Sub(p.lastDraw) < progressRefreshInterval {
		return
	}
	p.lastDraw = now

	percent := 0.0
	if p.total > 0 {
		percent = float64(p.transferred) / float64(p.total)
		if percent > 1 {
			percent = 1
		}
	} else if force {
		percent = 1
	}

	completed := int(percent * progressBarWidth)
	bar := strings.Repeat("=", completed) + strings.Repeat("-", progressBarWidth-completed)
	elapsed := now.Sub(p.startedAt).Seconds()
	var bytesPerSecond float64
	if elapsed > 0 {
		bytesPerSecond = float64(p.transferred) / elapsed
	}

	fmt.Fprintf(
		p.output,
		"\r%s [%s] %3.0f%% %s/%s %s/s\033[K",
		p.label,
		bar,
		percent*100,
		formatByteCount(p.transferred),
		formatByteCount(p.total),
		formatByteCount(int64(bytesPerSecond)),
	)
}

func newDirectoryTransferProgress(
	output io.Writer,
	label string,
	totalBytes int64,
	totalFiles int,
) *directoryTransferProgress {
	progress := &directoryTransferProgress{
		output:     output,
		label:      label,
		totalBytes: totalBytes,
		totalFiles: totalFiles,
		startedAt:  time.Now(),
		enabled:    isTerminalWriter(output),
	}
	progress.draw(true)
	return progress
}

func (p *directoryTransferProgress) Reader(reader io.Reader) io.Reader {
	return &directoryProgressReader{reader: reader, progress: p}
}

func (r *directoryProgressReader) Read(buffer []byte) (int, error) {
	n, err := r.reader.Read(buffer)
	if n > 0 {
		r.progress.transferred += int64(n)
		r.progress.draw(false)
	}
	return n, err
}

func (p *directoryTransferProgress) FileDone() {
	p.completedFiles++
	p.draw(false)
}

func (p *directoryTransferProgress) Finish() {
	if !p.enabled {
		return
	}
	p.draw(true)
	fmt.Fprintln(p.output)
}

func (p *directoryTransferProgress) draw(force bool) {
	if !p.enabled {
		return
	}
	now := time.Now()
	if !force && now.Sub(p.lastDraw) < progressRefreshInterval {
		return
	}
	p.lastDraw = now

	fraction := 1.0
	if p.totalBytes > 0 {
		fraction = float64(p.transferred) / float64(p.totalBytes)
	} else if p.totalFiles > 0 {
		fraction = float64(p.completedFiles) / float64(p.totalFiles)
	}
	if fraction > 1 {
		fraction = 1
	}
	completed := int(fraction * progressBarWidth)
	bar := strings.Repeat("=", completed) +
		strings.Repeat("-", progressBarWidth-completed)
	elapsed := now.Sub(p.startedAt).Seconds()
	var bytesPerSecond float64
	if elapsed > 0 {
		bytesPerSecond = float64(p.transferred) / elapsed
	}

	fmt.Fprintf(
		p.output,
		"\r%s [%s] %3.0f%% %d/%d files %s/%s %s/s\033[K",
		p.label,
		bar,
		fraction*100,
		p.completedFiles,
		p.totalFiles,
		formatByteCount(p.transferred),
		formatByteCount(p.totalBytes),
		formatByteCount(int64(bytesPerSecond)),
	)
}

func isTerminalWriter(output io.Writer) bool {
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func formatByteCount(value int64) string {
	const unit = 1024
	if value < unit {
		return fmt.Sprintf("%d B", value)
	}
	divisor, exponent := int64(unit), 0
	for quotient := value / unit; quotient >= unit && exponent < 4; quotient /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(value)/float64(divisor), "KMGTPE"[exponent])
}
