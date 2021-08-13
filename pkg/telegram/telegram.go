package telegram

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

type Bot struct {
	bot      *tb.Bot
	chat     *tb.Chat
	boot     time.Time
	messages chan string
}

func New(token string, chatID int) (*Bot, error) {
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("telegram: couldn't create bot: %w", err)
	}
	chat, err := b.ChatByID(strconv.Itoa(chatID))
	if err != nil {
		return nil, fmt.Errorf("telegram: couldn't create chat %d: %w", chatID, err)
	}
	bot := &Bot{
		bot:      b,
		chat:     chat,
		boot:     time.Now(),
		messages: make(chan string, 100),
	}
	return bot, nil
}

func (b *Bot) HandleChat(chatID int64, skipReply bool, handler func(string)) {
	b.bot.Handle(tb.OnText, func(m *tb.Message) {
		if m.Chat.ID != chatID && m.Chat.ID != b.chat.ID {
			return
		}
		if m.Time().Before(b.boot) {
			return
		}
		if m.IsReply() && skipReply {
			return
		}
		handler(m.Text)
	})
}

func (b *Bot) HandleCommand(command string, handler func(string)) {
	b.bot.Handle(fmt.Sprintf("/%s", command), func(m *tb.Message) {
		if m.Chat.ID != b.chat.ID {
			return
		}
		if m.Time().Before(b.boot) {
			return
		}
		handler(m.Payload)
	})
}

func (b *Bot) Run(ctx context.Context) error {
	go b.bot.Start()
	defer b.bot.Stop()
	defer b.bot.Send(b.chat, "🛑 bot stopping")
	var msg string
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg = <-b.messages:
		}
		opts := tb.ModeDefault
		if strings.Contains(msg, "`") {
			opts = tb.ModeMarkdown
		}
		if _, err := b.bot.Send(b.chat, msg, opts); err != nil {
			log.Println(err)
		}
		select {
		case <-ctx.Done():
			return nil
		// Wait to avoid rate limit errors
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (b *Bot) Print(v ...interface{}) {
	msg := fmt.Sprintln(v...)
	log.Print(msg)
	b.messages <- msg
}
