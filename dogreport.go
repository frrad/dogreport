package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/frrad/dogreport/wagapi"
	"github.com/frrad/settings"
)

type DogReportSettings struct {
	Username      string          `json:"username"`
	Password      string          `json:"password"`
	Token         string          `json:"token"`
	ReportedWalks map[string]bool `json:"reported_walks"`
}

func main() {
	set := DogReportSettings{
		Username: "username",
		Password: "password",
		Token:    "token",
	}
	setter, err := settings.NewSettings(&set, []string{"~/.dogreport"})
	if err != nil {
		panic(fmt.Errorf("Error parsing settings file: %v", err))
	}

	client := &http.Client{}
	var apiClient *wagapi.Client

	apiClient, err = wagapi.NewClientWithToken(client, set.Token)
	if err != nil {
		var token string
		apiClient, token, err = wagapi.NewClientWithUsernamePassword(
			client,
			set.Username,
			set.Password,
		)
		if err != nil {
			panic(err)
		}
		set.Token = token
		setter.Save()
	}

	// Now we should have a working client.

	allWalks := apiClient.LookupPastWalks()
	if set.ReportedWalks == nil {
		set.ReportedWalks = make(map[string]bool)
	}

	walksToReport := make(map[string]wagapi.Walk)
	walkers := make(map[int64]wagapi.Walker)
	for walkID, walk := range allWalks {
		if !set.ReportedWalks[walkID] {
			set.ReportedWalks[walkID] = true
			walksToReport[walkID] = walk

			walkerID := walk.WalkerID
			if _, ok := walkers[walkerID]; !ok {
				walkers[walkerID] = apiClient.LookupWalkerInt64(walkerID)
			}
		}
	}
	produceReport(walksToReport, walkers)
	setter.Save()
}

func printTable(data [][]string) string {
	ans := "<table>\n"

	for _, row := range data {
		ans += "<tr>\n"
		for _, datum := range row {
			ans += fmt.Sprintf("<td>%s</td>", datum)
		}
		ans += "\n</tr>\n"
	}

	ans += "</table>"

	return ans
}

func produceWalkReport(walk wagapi.Walk, walker wagapi.Walker) {
	tableContents := [][]string{}

	title := fmt.Sprintf("<b>%s</b>", walk.Date)
	tableContents = append(tableContents, []string{title})

	thumb := fmt.Sprintf("<img src=\"%s\" width=\"50px\">", walker.Thumb)
	completeCount := fmt.Sprintf("%d", walker.WalkCompletedCount)
	rating := fmt.Sprintf("%.3f", walker.Rating)
	data := [][]string{
		[]string{thumb, walker.FirstName, rating, completeCount},
	}
	tableContents = append(tableContents, []string{printTable(data)})

	photo := fmt.Sprintf(`<a href="%s"><img src="%s" width="600px"></a>`, walk.PhotoURL, walk.PhotoURL)
	mapx := fmt.Sprintf(`<a href="%s"><img src="%s" width="600px"></a>`, walk.WalkMap, walk.WalkMap)
	data = [][]string{
		[]string{photo},
		[]string{mapx},
	}
	tableContents = append(tableContents, []string{printTable(data)})

	locked, peed, pood, dist := strconv.Itoa(int(walk.IsDoorLocked)), strconv.Itoa(int(walk.IsPee)), strconv.Itoa(int(walk.IsPoo)), fmt.Sprintf("%.2f miles", walk.Distance)
	data = [][]string{
		[]string{"door locked", "peed", "pood", "distance"},
		[]string{locked, peed, pood, dist},
	}
	tableContents = append(tableContents, []string{printTable(data)})

	payout := fmt.Sprintf("%.2f", walk.Payout)
	tip := fmt.Sprintf("%.2f", walk.Tip)
	total := fmt.Sprintf("%.2f", walk.Total)
	data = [][]string{
		[]string{"payout", "tip", "total"},
		[]string{payout, tip, total},
	}
	tableContents = append(tableContents, []string{printTable(data)})

	subTitle := []string{}
	vals := []string{}
	for _, charge := range walk.Invoice.Charges {
		subTitle = append(subTitle, charge.Description)
		vals = append(vals, fmt.Sprintf("%.2f", charge.Amount))
	}
	data = [][]string{
		subTitle,
		vals,
	}
	tableContents = append(tableContents, []string{printTable(data)})

	data = [][]string{
		[]string{"scheduled start", "start", "end", "scheduled end"},
		[]string{walk.WalkStart, walk.WalkStarted, walk.WalkCompleted, walk.WalkEnd},
	}
	printTable(data)
	tableContents = append(tableContents, []string{printTable(data)})

	tableContents = append(tableContents, []string{walk.Note})
	fmt.Println(printTable(tableContents))
}

func produceReport(walks map[string]wagapi.Walk, walkers map[int64]wagapi.Walker) {
	if len(walks) == 0 {
		return
	}

	var walkIds []int
	for key := range walks {
		keyInt, _ := strconv.Atoi(key)
		walkIds = append(walkIds, keyInt)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(walkIds)))

	fmt.Println("<html><body>")
	fmt.Println("<style>table, th, td { border: 1px solid black; }</style>")

	for _, key := range walkIds {
		keyString := strconv.Itoa(key)
		walk := walks[keyString]
		walker := walkers[walk.WalkerID]
		produceWalkReport(walk, walker)
		fmt.Println("<hr>")
	}
	fmt.Println("</body></html>")

}
