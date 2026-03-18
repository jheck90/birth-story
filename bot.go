package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Discord interaction types
const (
	interactionPing           = 1
	interactionApplicationCmd = 2
)

// Discord response types
const (
	responsePong    = 1
	responseMessage = 4
)

const flagEphemeral = 64

type interaction struct {
	Type int              `json:"type"`
	Data *interactionData `json:"data"`
}

type interactionData struct {
	Name    string              `json:"name"`
	Options []interactionOption `json:"options"`
}

type interactionOption struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type interactionResponse struct {
	Type int                   `json:"type"`
	Data *interactionRespData  `json:"data,omitempty"`
}

type interactionRespData struct {
	Content string `json:"content"`
	Flags   int    `json:"flags,omitempty"`
}

func interactionsHandler(store *Store, publicKeyHex string) http.HandlerFunc {
	pubKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil || len(pubKeyBytes) != ed25519.PublicKeySize {
		panic(fmt.Sprintf("DISCORD_PUBLIC_KEY is invalid: %v", err))
	}
	pubKey := ed25519.PublicKey(pubKeyBytes)

	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		sig := r.Header.Get("X-Signature-Ed25519")
		ts := r.Header.Get("X-Signature-Timestamp")
		if !verifySignature(pubKey, sig, ts, body) {
			http.Error(w, "invalid request signature", http.StatusUnauthorized)
			return
		}

		var in interaction
		if err := json.Unmarshal(body, &in); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		switch in.Type {
		case interactionPing:
			json.NewEncoder(w).Encode(interactionResponse{Type: responsePong})

		case interactionApplicationCmd:
			if in.Data == nil || in.Data.Name != "update" {
				http.Error(w, "unknown command", http.StatusBadRequest)
				return
			}
			msg := optionValue(in.Data.Options, "message")
			if msg == "" {
				replyEphemeral(w, "Message cannot be empty.")
				return
			}
			store.Add(msg)
			replyEphemeral(w, fmt.Sprintf("Posted update: %s", msg))

		default:
			http.Error(w, "unknown interaction type", http.StatusBadRequest)
		}
	}
}

func replyEphemeral(w http.ResponseWriter, content string) {
	json.NewEncoder(w).Encode(interactionResponse{
		Type: responseMessage,
		Data: &interactionRespData{Content: content, Flags: flagEphemeral},
	})
}

func optionValue(opts []interactionOption, name string) string {
	for _, o := range opts {
		if o.Name == name {
			return o.Value
		}
	}
	return ""
}

func verifySignature(pubKey ed25519.PublicKey, sigHex, timestamp string, body []byte) bool {
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return false
	}
	msg := append([]byte(timestamp), body...)
	return ed25519.Verify(pubKey, msg, sig)
}

// registerCommand registers the /update slash command for a specific guild.
// Guild-scoped commands update instantly (vs global which take up to 1 hour).
func registerCommand(appID, guildID, token string) error {
	type option struct {
		Type        int    `json:"type"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Required    bool   `json:"required"`
	}
	type command struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Options     []option `json:"options"`
	}

	cmd := command{
		Name:        "update",
		Description: "Post a birth story update to the live page",
		Options: []option{
			{Type: 3, Name: "message", Description: "The update to share", Required: true},
		},
	}

	body, _ := json.Marshal(cmd)
	url := fmt.Sprintf("https://discord.com/api/v10/applications/%s/guilds/%s/commands", appID, guildID)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API %d: %s", resp.StatusCode, respBody)
	}
	return nil
}
