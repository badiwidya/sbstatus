package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"

	bl "github.com/badiwidya/brightctl/backlight"
	"github.com/fsnotify/fsnotify"
)

var (
	separator           bool
	separatorBlockWidth int
)

type status struct {
	Name                string `json:"name"`
	FullText            string `json:"full_text"`
	Markup              string `json:"markup"`
	Separator           bool   `json:"separator,omitempty"`
	SeparatorBlockWidth int    `json:"separator_block_width"`
}

func main() {
	flag.BoolVar(&separator, "separator", false, "Enable separator")
	flag.IntVar(&separatorBlockWidth, "separator-width", 17, "Specify separator width")

	flag.Parse()

	bri, err := bl.New("/sys/class/backlight")
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal("[ERROR]", err)
	}
	defer watcher.Close()

	err = watcher.Add(bri.BrightnessPath)
	if err != nil {
		log.Fatal("[ERROR]", err)
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	fmt.Println("{\"version\":1}")
	fmt.Println("[")

	printStatus(bri)

	for {
		select {
		case <-ticker.C:
			printStatus(bri)
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Write) {
				printStatus(bri)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("[ERROR]", err)
		}
	}
}

func printStatus(bri *bl.Backlight) {
	now := time.Now().Format("15:04:05")
	briText := "<span foreground=\"#fabd2f\">BRI</span> "

	briVal, err := bri.GetPercentage()
	if err != nil {
		briText += "ERR"
	} else {
		briText += fmt.Sprintf("%d%%", int(briVal*100))
	}

	volText := fmt.Sprintf("<span foreground=\"#fabd2f\">VOL</span> %s", getVolume())

	modules := []status{
		{Name: "volume", FullText: volText, Markup: "pango", Separator: separator, SeparatorBlockWidth: separatorBlockWidth},
		{Name: "brightness", FullText: briText, Markup: "pango", Separator: separator, SeparatorBlockWidth: separatorBlockWidth},
		{Name: "datetime", FullText: now, Markup: "none", Separator: separator, SeparatorBlockWidth: separatorBlockWidth},
	}

	jsonData, _ := json.Marshal(modules)
	fmt.Printf("%s,\n", jsonData)
}

func getVolume() string {
	cmd := exec.Command("wpctl", "get-volume", "@DEFAULT_SINK@")
	out, err := cmd.Output()
	if err != nil {
		log.Println("[ERROR]", err)
		return "ERR"
	}

	text := strings.TrimSpace(string(out))

	if strings.Contains(text, "MUTED") {
		return "MUTED"
	}

	parts := strings.Fields(text)
	if len(parts) < 2 {
		log.Println("[ERROR] output parts is less than 2")
		return "ERR"
	}

	volStr := parts[1]

	volFloat, err := strconv.ParseFloat(volStr, 64)
	if err != nil {
		log.Println("[ERROR]", err)
		return "ERR"
	}

	return fmt.Sprintf("%d%%", int(volFloat*100))
}
