package store

import (
	"encoding/base32"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/isacikgoz/dbcmp/internal/testlib"
	"github.com/pborman/uuid"
)

var (
	encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769").WithPadding(base32.NoPadding)
	emojis   = []string{":grinning:", ":slightly_smiling_face:", ":smile:", ":sunglasses:", ":innocent:", ":hugging_face:"}
	letters  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// newId creates a unique identifier
func newId() string {
	return encoding.EncodeToString(uuid.NewRandom())
}

// randomName creates a random name
func randomName() string {
	return string(letters[rand.Intn(len(letters))])
}

// randomSentences generates random string from the embedded words file.
func randomSentence() string {
	words, err := testlib.Assets().ReadFile(filepath.Join("text", "words"))
	if err != nil {
		panic(err)
	}

	r := rand.Intn(34) // Magic number
	if r <= 0 {
		return "🙂" // if there is nothing to say, an emoji worths for thousands
	}

	var withEmoji bool
	// 10% of the times we add an emoji to the message.
	if rand.Float64() < 0.10 {
		withEmoji = true
		r--
	}

	var random string
	for i := 0; i < r; i++ {
		n := rand.Int() % len(words)
		random += string(words[n]) + " "
	}

	if withEmoji {
		return random + emojis[rand.Intn(len(emojis))]
	}

	return random[:len(random)-1] + "."
}

func nowMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
