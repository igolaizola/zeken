package mtproto

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

type Listener interface {
	Listen(ctx context.Context, callback func(string)) error
}

type listener struct {
	id       int
	hash     string
	phone    string
	session  string
	fromID   int64
	log      func(v ...interface{})
	callback func(string)
	code     func(context.Context) string
}

func New(id int, hash, phone, session string, fromID int64, log func(v ...interface{}), callback func(string), code func(context.Context) string) *listener {
	return &listener{
		id:       id,
		hash:     hash,
		phone:    phone,
		session:  session,
		fromID:   fromID,
		log:      log,
		callback: callback,
		code:     code,
	}
}

func (l *listener) Listen(ctx context.Context) error {
	codePrompt := func(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
		code := l.code(ctx)
		return strings.TrimSpace(code), nil
	}

	// This will setup and perform authentication flow.
	flow := auth.NewFlow(
		auth.CodeOnly(l.phone, auth.CodeAuthenticatorFunc(codePrompt)),
		auth.SendCodeOptions{},
	)

	dispatcher := tg.NewUpdateDispatcher()

	client := telegram.NewClient(l.id, l.hash, telegram.Options{
		SessionStorage: &session.FileStorage{
			Path: l.session,
		},
		UpdateHandler: dispatcher,
	})

	return client.Run(ctx, func(ctx context.Context) error {
		if err := client.Auth().IfNecessary(ctx, flow); err != nil {
			return err
		}
		// Setting up handler for incoming message.
		dispatcher.OnNewMessage(func(ctx context.Context, entities tg.Entities, u *tg.UpdateNewMessage) error {
			m, ok := u.Message.(*tg.Message)
			if !ok || m.Out {
				// Outgoing message, not interesting.
				return nil
			}

			// Check if message is forwarded from target peer.
			peerID, err := fromPeer(m.PeerID)
			if err != nil {
				log.Println(err)
				return nil
			}
			if peerID != l.fromID {
				return nil
			}

			// Forward message to target peer.
			l.callback(m.Message)
			return nil
		})
		l.log("Listening for mtproto messages...")
		<-ctx.Done()
		return nil
	})
}

func fromPeer(p tg.PeerClass) (id int64, err error) {
	switch v := p.(type) {
	case *tg.PeerUser:
		return v.UserID, nil
	case *tg.PeerChannel:
		return v.ChannelID, nil
	case *tg.PeerChat:
		return v.ChatID, nil
	}
	return 0, fmt.Errorf("invalid peer: %T", p)
}
