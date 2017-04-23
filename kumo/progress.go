package kumo

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

func red(text string) string {
	return fmt.Sprintf("\033[31m%s\033[39m", text)
}

func green(text string) string {
	return fmt.Sprintf("\033[32m%s\033[39m", text)
}

func getTermColumns() (int, error) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	cleanedOut := strings.TrimSpace(string(out))
	splitOut := strings.Split(cleanedOut, " ")
	cols, err := strconv.Atoi(splitOut[1])
	if err != nil {
		return 0, err
	}

	return cols, nil
}

type ByteSize float64

const (
	_           = iota // ignore first value by assigning to blank identifier
	KB ByteSize = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
	ZB
	YB
)

func (b ByteSize) String() string {
	switch {
	case b >= YB:
		return fmt.Sprintf("%.2fYB", b/YB)
	case b >= ZB:
		return fmt.Sprintf("%.2fZB", b/ZB)
	case b >= EB:
		return fmt.Sprintf("%.2fEB", b/EB)
	case b >= PB:
		return fmt.Sprintf("%.2fPB", b/PB)
	case b >= TB:
		return fmt.Sprintf("%.2fTB", b/TB)
	case b >= GB:
		return fmt.Sprintf("%.2fGB", b/GB)
	case b >= MB:
		return fmt.Sprintf("%.2fMB", b/MB)
	case b >= KB:
		return fmt.Sprintf("%.2fKB", b/KB)
	}
	return fmt.Sprintf("%.2fB", b)
}

func secondsToHuman(interval int) string {
	seconds := interval % 60
	minutes := interval / 60

	switch {
	case minutes >= 60:
		hours := minutes / 60
		minutes = minutes % 60

		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	case minutes >= 1:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

const (
	PREFIX_DEFAULT         = "↳"
	PREFIX_PAR2            = "🔧 "
	PREFIX_COMPLETE_OK     = "✔"
	PREFIX_COMPLETE_BROKEN = "✘"
)

type Progress struct {
	Wait           *sync.WaitGroup
	Stop           chan bool
	brokenSize     int64
	currentSize    int64
	totalSize      int64
	brokenSegments int
	totalSegments  int
	mu             sync.Mutex
	start          int64
	prefix         string
}

func NewProgress() *Progress {
	return &Progress{
		Stop:   make(chan bool, 1),
		Wait:   new(sync.WaitGroup),
		prefix: PREFIX_DEFAULT,
	}
}

func (p *Progress) SetTotalSize(size int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalSize = size
}

func (p *Progress) Add(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.totalSegments += 1
	p.currentSize += bytes
}

func (p *Progress) addBroken(bytes int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.brokenSegments += 1
	p.brokenSize += bytes
}

func (p *Progress) isBroken() bool {
	return p.brokenSegments > 0
}

func (p *Progress) elapsed() int {
	elapsed := int(time.Now().Unix() - p.start)
	if elapsed == 0 {
		elapsed = 1
	}
	return elapsed
}

func (p *Progress) speed() float64 {
	elapsed := p.elapsed()
	return float64(p.currentSize) / float64(elapsed)
}

func (p *Progress) eta() int {
	if p.speed() == 0 {
		return 0
	} else {
		s := int(math.Ceil(float64(p.totalSize-p.currentSize) / p.speed()))
		return s
	}
}

func (p *Progress) etaString() string {
	eta := p.eta()
	if eta == 0 {
		return "∞"
	}
	return secondsToHuman(eta)
}

func (p *Progress) percentage() string {
	percentage := 0.0
	if p.totalSize > 0 {
		percentage = (float64(p.currentSize) / float64(p.totalSize))
	}
	return fmt.Sprintf("%.1f%%", percentage*100)
}

func (p *Progress) printBroken() {
	// ✘ 524.25KB/524.25KB 87.37KB/s 100% ↯ 6s
	// ┗━➤ 1.2MB/3.4MB (2/3) segments broken!
	suffix := ""
	if p.totalSegments > 1 {
		suffix = "s"
	}
	fmt.Printf("\n%s %s/%s (%d/%d) segment%s broken!", red("┗━➤"), ByteSize(p.brokenSize).String(), ByteSize(p.totalSize).String(), p.brokenSegments, p.totalSegments, suffix)
}

func (p *Progress) printProgress(prefix, currentSize, total, speed, percent, separator, _time string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cols, err := getTermColumns()
	if err != nil {
		println(err)
	}

	progress := fmt.Sprintf("%s %s/%s %s/s %s %s %s", prefix, currentSize, total, speed, percent, separator, _time)
	padding := strings.Repeat(" ", cols-len(progress))

	fmt.Print("\r", progress, padding)
}

func (p *Progress) reset() {
	p.brokenSize = 0
	p.currentSize = 0
	p.brokenSegments = 0
	p.totalSegments = 0
	p.start = 0
}

func (p *Progress) Run() {
	p.Wait.Add(1)
	defer p.Wait.Done()

	p.start = time.Now().Unix()

	for {
		select {
		case <-p.Stop:
			return
		default:
		}

		p.mu.Lock()
		totalSize := p.totalSize
		currentSize := p.currentSize
		p.mu.Unlock()

		if totalSize > 0 && currentSize >= totalSize {
			total := ByteSize(totalSize).String()
			prefix := green(PREFIX_COMPLETE_OK)
			if p.isBroken() {
				prefix = red(PREFIX_COMPLETE_BROKEN)
			}
			// ✔ 396.86KB/396.86KB 30.53KB/s 100% ↯ 32s
			p.printProgress(prefix, total, total, ByteSize(p.speed()).String(), "100%", "↯", secondsToHuman(p.elapsed()))
			if p.isBroken() {
				p.printBroken()
			}
			fmt.Println()
			return
		}

		prefix := p.prefix
		if p.isBroken() {
			prefix = fmt.Sprintf("%s (%d/%d)", red(prefix), p.brokenSegments, p.totalSegments)
		}

		// ↳ 146.92KB/396.86KB 13.36KB/s 37.0% ↦ 19s
		p.printProgress(prefix, ByteSize(currentSize).String(), ByteSize(totalSize).String(), ByteSize(p.speed()).String(), p.percentage(), "↦", p.etaString())

		time.Sleep(1 * time.Second)
	}
}
