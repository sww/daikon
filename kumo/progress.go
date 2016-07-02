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

type Progress struct {
	Current int
	Total   int
	Wait    *sync.WaitGroup
	Stop    chan bool
	mu      sync.Mutex
	start   int64
}

func InitProgress() *Progress {
	return &Progress{
		Stop: make(chan bool, 1),
		Wait: new(sync.WaitGroup),
	}
}

func (p *Progress) Add(bytes int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Current += bytes
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
	return float64(p.Current) / float64(elapsed)
}

func (p *Progress) eta() int {
	if p.speed() == 0 {
		return 0
	} else {
		s := int(math.Ceil(float64(p.Total-p.Current) / p.speed()))
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
	return fmt.Sprintf("%.1f%%", (float64(p.Current)/float64(p.Total))*100)
}

func (p *Progress) printProgress(prefix, current, total, speed, percent, separator, _time string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cols, err := getTermColumns()
	if err != nil {
		println(err)
	}

	progress := fmt.Sprintf("%s %s/%s %s/s %s %s %s", prefix, current, total, speed, percent, separator, _time)
	padding := strings.Repeat(" ", cols-len(progress))

	fmt.Print("\r", progress, padding)
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

		if p.Total > 0 && p.Current >= p.Total {
			total := ByteSize(p.Total).String()
			// ✔ 396.86KB/396.86KB 30.53KB/s 100% ↯ 32s
			p.printProgress("✔", total, total, ByteSize(p.speed()).String(), "100%", "↯", secondsToHuman(p.elapsed()))
			fmt.Println()
			return
		}

		// ↳ 146.92KB/396.86KB 13.36KB/s 37.0% ↦ 19s
		p.printProgress("↳", ByteSize(p.Current).String(), ByteSize(p.Total).String(), ByteSize(p.speed()).String(), p.percentage(), "↦", p.etaString())

		time.Sleep(1 * time.Second)
	}
}
