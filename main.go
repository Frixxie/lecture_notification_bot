package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"fasteraune.com/calendar_util"
	"github.com/bwmarrin/discordgo"
)

var BotId string
var Events []calendar_util.CsvEvent
var NotifyChannel string
var StackOfEvents *Stack

type StackNode struct {
	Event calendar_util.CsvEvent
	Next  *StackNode
}

type Stack struct {
	Top *StackNode
}

func (s *Stack) Push(event calendar_util.CsvEvent) {
	node := &StackNode{
		Event: event,
		Next:  s.Top,
	}
	s.Top = node
}

func NewStack() *Stack {
	return &Stack{}
}

func convertToStack(events []calendar_util.CsvEvent) *Stack {
	stack := new(Stack)
	for i := len(events) - 1; i >= 0; i-- {
		stack.Push(events[i])
	}
	return stack
}

func (s *Stack) Pop() *StackNode {
	if s.Top == nil {
		return nil
	}
	node := s.Top
	s.Top = s.Top.Next
	return node
}

func WhenEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	fmt.Println(m.Content)
	fmt.Println(m.Author.ID)
	fmt.Println(m.Author.Username)

	if m.Content == "!event" && m.ChannelID == NotifyChannel {
		s.ChannelMessageSend(m.ChannelID, calendar_util.NextCsvEvent(Events).String())
	}
}

func Help(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == "!help" && m.ChannelID == NotifyChannel {
		s.ChannelMessageSend(m.ChannelID, "This is lecture_notification bot, It will notify a channel about lectures and other university related events\nCommands:\n!event - Shows next event\n!help - Shows this message\n")
	}
}

func NotifyEvent(s *discordgo.Session, r *discordgo.Ready) {
	var event *calendar_util.CsvEvent
	for {
		if event == nil {
			event = &StackOfEvents.Pop().Event
			// if the stack is empty, we are done
			if event == nil {
				s.ChannelMessageSend(NotifyChannel, "No events left service is now stopped")
				fmt.Println("No events left service is now stopped")
				return
			}
			fmt.Printf("Event:\n%s\n", (*event).String())
		}
		timeNow := time.Now()
		timeUntilEvent := event.DtStart.Sub(timeNow)
		if timeUntilEvent < time.Minute*15 {
			s.ChannelMessageSend(NotifyChannel, event.String())
			//i suppose the gc will take care of the memory
			event = nil
		}
		//To lessen the load on the server, we sleep for a minute
		time.Sleep(time.Minute)
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	token := "NTgwNDYyMjU2Mzk1OTExMTc3.XORDmg.4vplGl3G_bsEjmGukq0ppKBogyw"
	NotifyChannel = "922805648251580416"
	LecNotBot, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	user, err := LecNotBot.User("@me")
	if err != nil {
		panic(err)
	}

	BotId = user.ID

	urls := []string{"https://tp.uio.no/uit/timeplan/excel.php?type=course&sort=week&id[]=INF-3203%2C1&id[]=INF-3701%2C1", "https://tp.uio.no/ntnu/timeplan/excel.php?type=courseact&id%5B%5D=GEOG2023%C2%A4&id%5B%5D=KULMI2710%C2%A4&sem=22v&stop=1"}
	csv, err := calendar_util.ReadCsvEvents(urls)
	if err != nil {
		fmt.Println(err)
		return
	}
	Events = csv
	sort.Slice(csv, func(i, j int) bool {
		return csv[i].DtStart.Before(csv[j].DtStart.Time)
	})
	StackOfEvents = convertToStack(Events)

	LecNotBot.AddHandler(WhenEvent)
	LecNotBot.AddHandler(Help)
	LecNotBot.AddHandler(NotifyEvent)

	err = LecNotBot.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	fmt.Println("Closing bot...")
	LecNotBot.Close()
}
