package main

import (
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"fasteraune.com/calendar_util"
	"github.com/bwmarrin/discordgo"
)

var (
	BotId         string
	Events        []calendar_util.CsvEvent
	NotifyChannel string
	StackOfEvents []*Stack
)

type StackNode struct {
	Event calendar_util.CsvEvent
	Next  *StackNode
}

type Stack struct {
	Top     *StackNode
	Channel string
	Owner   string
	Len     int
	Active  bool
	Rw      *sync.RWMutex
}

func (s *Stack) Push(event calendar_util.CsvEvent) {
	node := &StackNode{
		Event: event,
		Next:  s.Top,
	}
	s.Top = node
	s.Len += 1
}

func NewStack(channel string, owner string) *Stack {
	return &Stack{
		Top:     nil,
		Channel: channel,
		Owner:   owner,
		Len:     0,
		Active:  true,
		Rw:      &sync.RWMutex{},
	}
}

func ConvertToStack(events []calendar_util.CsvEvent, channel string, owner string) *Stack {
	stack := NewStack(channel, owner)
	for i := len(events) - 1; i >= 0; i-- {
		stack.Push(events[i])
		println(stack.Len)
	}
	return stack
}

func (s Stack) String() string {
	return fmt.Sprintf("%s, %s, %d events, Active %t", s.Owner, s.Channel, s.Len, s.Active)
}

func (s *Stack) Pop() *calendar_util.CsvEvent {
	if s.Top == nil {
		return nil
	}
	node := s.Top
	s.Top = s.Top.Next
	s.Len -= 1
	return &node.Event
}

// TODO: rewrite me!
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
		s.ChannelMessageSend(m.ChannelID, "This is notification bot, It will notify a channel about lectures and other university related events\nCommands:\n!event - Shows next event\n!help - Shows this message\n")
	}
}

func Join(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	content := strings.Split(m.Content, " ")
	urls := content[1:]

	if content[0] != "!join" {
		return
	}

	if len(content) < 2 {
		s.ChannelMessageSend(m.ChannelID, "Usage: !join <urls> ")
		return
	}

	csv, err := calendar_util.ReadCsvEvents(urls)
	if err != nil {
		fmt.Println(err)
		return
	}
	sort.Slice(csv, func(i, j int) bool {
		return csv[i].DtStart.Before(csv[j].DtStart.Time)
	})
	stack := ConvertToStack(csv, m.ChannelID, m.Author.Username)
	StackOfEvents = append(StackOfEvents, stack)
	go notify_events(&s, &stack)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Thank you @%s for adding lecture notification bot. Service is now started, %d events will be notified", stack.Owner, stack.Len))
}

func Leave(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == "!leave" {
		for i := 0; i < len(StackOfEvents); i++ {
			if StackOfEvents[i].Channel == m.ChannelID {
				// TODO: remove the stack from the list
				StackOfEvents[i].Rw.Lock()
				StackOfEvents[i].Active = false
				StackOfEvents[i].Rw.Unlock()
				StackOfEvents = append(StackOfEvents[:i], StackOfEvents[i+1:]...)
				// s.ChannelMessageSend(m.ChannelID, "Service is now stopped")
				return
			}
		}
	}
}

func notify_events(s **discordgo.Session, stackofevents **Stack) {
	var event *calendar_util.CsvEvent
	session := *s
	stack := *stackofevents
	for {
		stack.Rw.RLock()
		fmt.Println(stack)
		if !stack.Active {
			session.ChannelMessageSend(stack.Channel, fmt.Sprintf("Service for Owner %s is now stopped", stack.Owner))
			fmt.Println("Service is now stopped")
			return
		}
		if event == nil {
			event = stack.Pop()
			// if the stack is empty, we are done
			if event == nil {
				session.ChannelMessageSend(stack.Channel, "No events left service is now stopped")
				fmt.Println("No events left service is now stopped")
				return
			}
			fmt.Printf("Event:\n%s\n", (*event).String())
		}
		timeUntilEvent := event.DtStart.Sub(time.Now())
		if timeUntilEvent < time.Minute*15 {
			session.ChannelMessageSend(NotifyChannel, fmt.Sprintf("Hey %s, next event is:\n%s", stack.Owner, event.String()))
			//i suppose the gc will take care of the memory
			event = nil
		}
		//To lessen the load on the server, we sleep for a minute
		stack.Rw.RUnlock()
		time.Sleep(time.Second * 5)
	}
}

//// TODO: rewrite me!
//func NotifyEvent(s *discordgo.Session, r *discordgo.Ready) {
//	var event *calendar_util.CsvEvent
//	for {
//		if event == nil {
//			event = StackOfEvents.Pop()
//			// if the stack is empty, we are done
//			if event == nil {
//				s.ChannelMessageSend(NotifyChannel, "No events left service is now stopped")
//				fmt.Println("No events left service is now stopped")
//				return
//			}
//			fmt.Printf("Event:\n%s\n", (*event).String())
//		}
//		timeUntilEvent := event.DtStart.Sub(time.Now())
//		if timeUntilEvent < time.Minute*15 {
//			s.ChannelMessageSend(NotifyChannel, event.String())
//			//i suppose the gc will take care of the memory
//			event = nil
//		}
//		//To lessen the load on the server, we sleep for a minute
//		time.Sleep(time.Minute)
//	}
//}

func main() {
	// Create a new Discord session using the provided bot token.
	token := "NTgwNDYyMjU2Mzk1OTExMTc3.XORDmg.4vplGl3G_bsEjmGukq0ppKBogyw"
	// NotifyChannel = "922805648251580416"
	LecNotBot, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	user, err := LecNotBot.User("@me")
	if err != nil {
		panic(err)
	}

	BotId = user.ID

	// urls := []string{"https://tp.uio.no/uit/timeplan/excel.php?type=course&sort=week&id[]=INF-3203%2C1&id[]=INF-3701%2C1", "https://tp.uio.no/ntnu/timeplan/excel.php?type=courseact&id%5B%5D=GEOG2023%C2%A4&id%5B%5D=KULMI2710%C2%A4&sem=22v&stop=1"}
	// csv, err := calendar_util.ReadCsvEvents(urls)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// Events = csv
	// sort.Slice(csv, func(i, j int) bool {
	// 	return csv[i].DtStart.Before(csv[j].DtStart.Time)
	// })
	// StackOfEvents[0] = ConvertToStack(Events, NotifyChannel, "fredrik")
	// LecNotBot.AddHandler(NotifyEvent)

	LecNotBot.AddHandler(WhenEvent)
	LecNotBot.AddHandler(Help)
	LecNotBot.AddHandler(Join)
	LecNotBot.AddHandler(Leave)

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
