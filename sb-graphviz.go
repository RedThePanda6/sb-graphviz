package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	actionsFile = flag.String(
		"actionsfile",
		"D:\\Streamer.bot\\prime\\data\\actions.json",
		"Path to SB actions.json",
	)
	outFile = flag.String(
		"outfile",
		"G:\\My Drive\\Streaming\\Chatbot\\StreamerBot\\sb.dot",
		"Path to Graphviz file",
	)
	emptyAction = "00000000-0000-0000-0000-000000000000"
	// Sub-action types that don't actually run other actions.
	skipArrows = map[int]bool{
		1004: true,
	}
	// Only render triggers that we've seen.
	triggersSeen = map[int]bool{}
	// Generic mapping of triggers.
	// This is really ugly.
	triggers = map[int]string{
		101:   "Follow",
		102:   "Cheer",
		103:   "Subscription",
		104:   "Resubscription",
		105:   "Gift Subscription",
		106:   "Gift Bomb",
		107:   "Raid",
		108:   "Hype Train Start",
		110:   "Hype Train Level Up",
		111:   "Hype Train End",
		112:   "Reward Redemption",
		116:   "Community Goal Contribution",
		118:   "Stream Update",
		120:   "First Words",
		121:   "Sub Counter Rollover",
		127:   "Poll Completed",
		130:   "Prediction Completed",
		133:   "Chat Message",
		135:   "Chat Message Deleted",
		136:   "User Timed Out",
		137:   "User Banned (T)",
		139:   "Ad Run",
		14003: "OBS Event",
		14004: "OBS Scene Changed",
		154:   "Stream Online",
		155:   "Stream Offline",
		158:   "Raid Start",
		159:   "Raid Send",
		161:   "Poll Terminated",
		186:   "Upcoming Ad",
		189:   "VIP Added",
		190:   "VIP Removed",
		29003: "Remote Instance Trigger",
		32004: "Group User Added",
		32005: "Group User Removed",
		4001:  "Broadcast Started",
		4002:  "Broadcast Ended",
		4003:  "Message (YT)",
		4005:  "User Banned (YT)",
		401:   "Command Triggered",
		4016:  "First Words (YT)",
		4018:  "New Subscriber",
		463:   "AutoMod Message Held",
		474:   "Shared Chat Session Begin",
		476:   "Shared Chat Session End",
		477:   "Prime Paid Upgrade",
		478:   "Pay It Forward",
		479:   "Gift Paid Upgrade",
		501:   "File Changed",
		601:   "Quote Added",
		602:   "Show Quote",
		701:   "Timed Actions",
		702:   "Test",
		706:   "Streamer.Bot Started",
		709:   "Global Variable Updated",
	}
)

// StreamerBot actions.json structs.
type data struct {
	Actions []action `json:"actions"`
}

type action struct {
	ActionId           string   `json:"actionId"`
	Actions            []action `json:"actions"`
	ElseActionId       string   `json:"elseActionId"`
	ElseRunImmediately bool     `json:"elseRunImmediately"`
	Group              string   `json:"group"`
	Id                 string   `json:"id"`
	Name               string   `json:"name"`
	Queue              string   `json:"queue"`
	// This is misspelled in the configs. FML.
	RunImmedately  bool      `json:"runImmedately"`
	RunImmediately bool      `json:"runImmediately"`
	Triggers       []trigger `json:"triggers"`
	Type           int       `json:"type"`
}

type trigger struct {
	Id   string `json:"id"`
	Type int    `json:"type"`
}

// Graphviz structs
type subgraph struct {
	Name  string
	Label string
	Nodes []string
}

func readFromFile(f string) data {
	d := data{}
	file, err := os.Open(f)
	defer file.Close()
	if err != nil {
		fmt.Println("Error loading config:", err)
	}
	jsonParser := json.NewDecoder(file)
	jsonParser.Decode(&d)
	return d
}

func writeGraphviz(f string, n []string, s []subgraph, a []string) {
	outputFile, err := os.Create(f)
	if err != nil {
		fmt.Println("Error creating Graphviz file:", err)
	}
	defer outputFile.Close()

	// File header
	outputFile.WriteString("digraph streamerbot {\n")
	outputFile.WriteString("  rankdir = \"LR\"\n")
	outputFile.WriteString("\n")

	// Nodes
	for _, line := range n {
		outputFile.WriteString(fmt.Sprintf("%s\n", line))
	}
	for x, line := range triggers {
		if triggersSeen[x] {
			outputFile.WriteString(fmt.Sprintf("  \"%d\" [label=\"%s\" shape=diamond]\n", x, line))
		}
	}

	outputFile.WriteString("\n")

	// Subgraphs
	for _, g := range s {
		outputFile.WriteString(fmt.Sprintf("  subgraph %s {\n", g.Name))
		outputFile.WriteString(fmt.Sprintf("    label = \"%s\";\n", g.Label))
		for _, n := range g.Nodes {
			outputFile.WriteString(fmt.Sprintf("    %s;\n", n))
		}
		outputFile.WriteString("  }\n")
		outputFile.WriteString("\n")
	}

	// Arrows
	for _, line := range a {
		outputFile.WriteString(fmt.Sprintf("%s\n", line))
	}

	// File footer
	outputFile.WriteString("}\n")

	outputFile.Sync()

	w := bufio.NewWriter(outputFile)
	w.Flush()
}

func generateSubgraphs(d data) []subgraph {
	// Build list of all groups.
	groups := map[string]bool{}

	for _, a := range d.Actions {
		if !groups[a.Group] {
			groups[a.Group] = true
		}
	}

	subgraphs := []subgraph{}
	x := 0

	for g := range groups {
		// Build list of nodes
		nodes := []string{}
		for _, a := range d.Actions {
			if a.Group == g {
				nodes = append(nodes, fmt.Sprintf("\"%s\"", a.Id))
			}
		}

		// Add to subgraphs
		subgraphs = append(subgraphs,
			subgraph{
				Name:  fmt.Sprintf("cluster_%d", x),
				Label: g,
				Nodes: nodes,
			},
		)
		x++
	}

	return subgraphs
}

func generateNodesLabels(d data) []string {
	nodes := []string{}

	for _, a := range d.Actions {
		nodes = append(nodes, fmt.Sprintf("  \"%s\" [label=\"%s\" shape=box style=rounded]", a.Id, a.Name))
	}

	return nodes
}

func generateArrows(d data) []string {
	arrows := []string{}

	for _, b := range d.Actions {
		for _, a := range b.Actions {
			// Skip actions that don't run other actions.
			// This includes things like Set Action State.
			if skipArrows[a.Type] {
				continue
			}

			if a.ActionId != emptyAction && a.ActionId != "" {
				line := fmt.Sprintf("  \"%s\" -> \"%s\" [color=blue]", b.Id, a.ActionId)

				if !a.RunImmedately && !a.RunImmediately {
					line = strings.Replace(line, "]", " style=dashed]", -1)
				}

				arrows = append(
					arrows,
					line,
				)
			}

			if a.ElseActionId != emptyAction && a.ElseActionId != "" {
				line := fmt.Sprintf("  \"%s\" -> \"%s\" [color=red]", b.Id, a.ElseActionId)

				if !a.ElseRunImmediately {
					line = strings.Replace(line, "]", " style=dashed]", -1)
				}

				arrows = append(
					arrows,
					line,
				)
			}
		}

		for _, t := range b.Triggers {
			arrows = append(
				arrows,
				fmt.Sprintf("  \"%d\" -> \"%s\" [color=green]", t.Type, b.Id),
			)

			if !triggersSeen[t.Type] {
				triggersSeen[t.Type] = true
			}
		}
	}

	return arrows
}

func main() {
	flag.Parse()

	data := readFromFile(*actionsFile)

	nodes := generateNodesLabels(data)
	subgraphs := generateSubgraphs(data)
	arrows := generateArrows(data)

	writeGraphviz(*outFile, nodes, subgraphs, arrows)
}
