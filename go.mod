module fasteraune.com/lecture_notification_bot

go 1.18

require (
	fasteraune.com/calendar_util v0.0.0-00010101000000-000000000000
	github.com/bwmarrin/discordgo v0.23.2
)

require (
	github.com/erizocosmico/go-ics v0.0.0-20180527181030-697b9000b86f // indirect
	github.com/gocarina/gocsv v0.0.0-20211203214250-4735fba0c1d9 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	golang.org/x/crypto v0.0.0-20220112180741-5e0467b6c7ce // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
)

replace fasteraune.com/calendar_util => ./uit_calendar_util/calendar_util
