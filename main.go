package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"fasteraune.com/uit_calendar_util"
	"github.com/bwmarrin/discordgo"
)

var BotId string
var Events []uit_calendar_util.Event

func notify_lecture(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	fmt.Println(m.Content)
	fmt.Println(m.Author.ID)
	fmt.Println(m.Author.Username)
	fmt.Println("replying with:\n", uit_calendar_util.NextEvent(Events))

	if m.Content == "!lec" && m.ChannelID == "922805648251580416" {
		s.ChannelMessageSend(m.ChannelID, uit_calendar_util.NextEvent(Events).String())
	}
}

func main() {
	// Create a new Discord session using the provided bot token.
	token := "NTgwNDYyMjU2Mzk1OTExMTc3.XORDmg.4vplGl3G_bsEjmGukq0ppKBogyw"
	LecNotBot, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}

	user, err := LecNotBot.User("@me")
	if err != nil {
		panic(err)
	}

	BotId = user.ID

	courses := []string{"INF-3203-1", "INF-3701-1"}
	url := uit_calendar_util.ConsructUrl("https://timeplan.uit.no/calendar.ics?sem=22v", courses)
	res, err := uit_calendar_util.GetData(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	Events = res

	LecNotBot.AddHandler(notify_lecture)

	err = LecNotBot.Open()
	if err != nil {
		panic(err)
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	LecNotBot.Close()
}
