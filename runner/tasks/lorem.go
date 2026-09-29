package tasks

import (
	"context"
	"math/rand/v2"
	"strings"
)

const (
	defaultLoremWords = 200
	loremMinSentences = 4
	loremMaxSentences = 8
	loremMinWords     = 6
	loremMaxWords     = 14
)

const loremOpening = "Lorem ipsum dolor sit amet, consectetur adipiscing elit."

var loremWords = strings.Fields(`
	lorem ipsum dolor sit amet consectetur adipiscing elit sed do eiusmod tempor
	incididunt ut labore et dolore magna aliqua enim ad minim veniam quis nostrud
	exercitation ullamco laboris nisi aliquip ex ea commodo consequat duis aute irure
	in reprehenderit voluptate velit esse cillum fugiat nulla pariatur excepteur sint
	occaecat cupidatat non proident sunt culpa qui officia deserunt mollit anim id est
	laborum curabitur pretium tincidunt lacus nunc pulvinar sapien ligula ornare
	integer vitae justo eget magna fermentum iaculis faucibus viverra`)

// LoremIpsum generates in.Words (default defaultLoremWords) words of random Lorem Ipsum,
// logging one paragraph per line.
func LoremIpsum(ctx context.Context, in Input, logf Logf) error {
	n := in.Words
	if n <= 0 {
		n = defaultLoremWords
	}
	return generateLorem(ctx, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())), n, logf)
}

func generateLorem(ctx context.Context, rng *rand.Rand, totalWords int, logf Logf) error {
	logf("Generating Lorem Ipsum (%d words)...", totalWords)

	opening := strings.Fields(strings.TrimSuffix(loremOpening, "."))
	remaining := totalWords
	for p := 1; remaining > 0; p++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		n := loremMinSentences + rng.IntN(loremMaxSentences-loremMinSentences+1)
		sentences := make([]string, 0, n)
		for ; n > 0 && remaining > 0; n-- {
			var words []string
			if p == 1 && len(sentences) == 0 {
				words = opening[:min(len(opening), remaining)]
			} else {
				words = loremSentence(rng, min(loremMinWords+rng.IntN(loremMaxWords-loremMinWords+1), remaining))
			}
			sentences = append(sentences, strings.TrimSuffix(strings.Join(words, " "), ",")+".")
			remaining -= len(words)
		}
		logf("Paragraph %d: %s", p, strings.Join(sentences, " "))
	}

	logf("Generated %d words", totalWords)
	logf("Lorem Ipsum generation completed")
	return nil
}

// loremSentence returns n random words, capitalized, with occasional commas.
func loremSentence(rng *rand.Rand, n int) []string {
	words := make([]string, n)
	for i := range n {
		w := loremWords[rng.IntN(len(loremWords))]
		if i == 0 {
			w = strings.ToUpper(w[:1]) + w[1:]
		}
		// Occasional comma, never right before the final word.
		if i > 1 && i < n-2 && rng.IntN(8) == 0 {
			w += ","
		}
		words[i] = w
	}
	return words
}
