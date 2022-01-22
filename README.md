# lecture notification bot

This to lecture notification bot, a discord bot designed to provide notification when the next lecture or university related event is due.
This functions on all Norwegian calendars which can be exported as excel format (it is really csv, they call it excel).

## how does it work?

It is designed with a pub-sub architecture.

It works by adding the bot to the server. then on the channel which you want the bot to be in call !join <url_to_the_csv_file>.
You can construct the url by going [here](https://tp.uio.no), if you want another university than UiO you can add for example NTNU in the url path.
