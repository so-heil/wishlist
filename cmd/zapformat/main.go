package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

type log map[string]any

const microToMiliRatio = 1000

func main() {
	r := os.Stdin

	scn := bufio.NewScanner(r)
	s := new(strings.Builder)
	for scn.Scan() {
		s.Reset()
		var l log
		if err := json.Unmarshal(scn.Bytes(), &l); err != nil {
			fmt.Println(scn.Text())
			continue
		}

		tf, ok := l["ts"].(float64)
		if !ok {
			fmt.Printf("format: cannot convert ts %s\n", l["ts"])
			continue
		}
		t := time.UnixMilli(int64(tf * microToMiliRatio))
		s.WriteString(color.BlueString("%s\t", t.Format(time.RFC3339)))

		level, ok := l["level"].(string)
		if !ok {
			fmt.Printf("format: cannot convert level %s to string\n", l["level"])
			continue
		}
		s.WriteString(color.MagentaString("%s\t", strings.ToUpper(level)))

		s.WriteString(color.HiYellowString("%s\t", l["msg"]))

		for key, val := range l {
			switch key {
			case "ts", "level", "caller", "msg":
				continue
			default:
				fmt.Fprintf(s, "%s[%s] ", color.HiGreenString("%s", key), color.HiCyanString("%s", val))
			}
		}

		s.WriteString(color.CyanString("%s\t", l["caller"]))

		fmt.Println(s.String())
	}
}
