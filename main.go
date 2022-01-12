package main

import (
	"fmt"
	"io/ioutil"
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
	}
	return stack
}

func (s Stack) String() string {
	s.Rw.RLock()
	res := fmt.Sprintf("%s, %s, %d events, Active %t", s.Owner, s.Channel, s.Len, s.Active)
	s.Rw.RUnlock()
	return res
}

func (s *Stack) Push(event calendar_util.CsvEvent) {
	s.Rw.Lock()
	node := &StackNode{
		Event: event,
		Next:  s.Top,
	}
	s.Top = node
	s.Len += 1
	s.Rw.Unlock()
}

func (s *Stack) Pop() *calendar_util.CsvEvent {
	s.Rw.Lock()
	if s.Top == nil {
		s.Rw.Unlock()
		return nil
	}
	node := s.Top
	s.Top = s.Top.Next
	s.Len -= 1
	s.Rw.Unlock()
	return &node.Event
}

func (s *Stack) Peek() *calendar_util.CsvEvent {
	s.Rw.RLock()
	if s.Top == nil {
		s.Rw.RUnlock()
		return nil
	}
	res := s.Top.Event
	s.Rw.RUnlock()
	return &res
}

func (s *Stack) SetState(active bool) {
	s.Rw.Lock()
	s.Active = active
	s.Rw.Unlock()
}

func (s *Stack) GetState() bool {
	s.Rw.RLock()
	active := s.Active
	s.Rw.RUnlock()
	return active
}

func WhenEvent(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == "!event" {
		for _, stack := range StackOfEvents {
			if stack.Channel == m.ChannelID {
				s.ChannelMessageSend(m.ChannelID, stack.Peek().String())
			}
		}
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

	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Thank you %s for adding lecture notification bot. Service is now started, %d events will be notified", stack.Owner, stack.Len))
	notify_events(&s, stack)
}

func Leave(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == "!leave" {
		for i := 0; i < len(StackOfEvents); i++ {
			if StackOfEvents[i].Channel == m.ChannelID && StackOfEvents[i].Owner == m.Author.Username {
				StackOfEvents[i].SetState(false)
				StackOfEvents = append(StackOfEvents[:i], StackOfEvents[i+1:]...)
			}
		}
	}
}

func notify_events(s **discordgo.Session, stack *Stack) {
	var event *calendar_util.CsvEvent
	session := *s
	for {
		fmt.Println(stack)
		//If the stack is deactivated, stop the loop
		if !stack.GetState() {
			session.ChannelMessageSend(stack.Channel, "Service for Owner is now stopped")
			fmt.Println("Service is now stopped", stack)
			return
		}
		if event == nil {
			event = stack.Peek()
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
			if timeUntilEvent > 0 {
				session.ChannelMessageSend(stack.Channel, event.String())
			}
			//i suppose the gc will take care of the memory
			event = stack.Pop()
			event = nil
		}
		//To lessen the load on the server, we sleep for a minute
		time.Sleep(time.Second * 5)
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	token, err := ioutil.ReadFile("bot_token")
	if err != nil {
		fmt.Println("Error reading token file")
		return
	}
	LecNotBot, err := discordgo.New("Bot " + strings.Trim(string(token), "\n"))
	if err != nil {
		panic(err)
	}

	user, err := LecNotBot.User("@me")
	if err != nil {
		panic(err)
	}

	BotId = user.ID

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
