/**************************************************************************
 * DiffScope Synthesis Platform                                           *
 * Copyright (C) 2026 Team OpenVPI                                        *
 *                                                                        *
 * This program is free software: you can redistribute it and/or modify   *
 * it under the terms of the GNU General Public License as published by   *
 * the Free Software Foundation, either version 3 of the License, or      *
 * (at your option) any later version.                                    *
 *                                                                        *
 * This program is distributed in the hope that it will be useful,        *
 * but WITHOUT ANY WARRANTY; without even the implied warranty of         *
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the          *
 * GNU General Public License for more details.                           *
 *                                                                        *
 * You should have received a copy of the GNU General Public License      *
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. *
 **************************************************************************/

package builder

import (
	"fmt"
	"math"

	"diffscope-synthesis-platform/internal/dsinfer"
)

const (
	spPronunciation   = "SP"
	apPronunciation   = "AP"
	slurPronunciation = "-"
)

type Note struct {
	Gap           float64
	Duration      float64
	Cents         int
	Pronunciation string
	Language      string
	Phonemes      []Phoneme
}

type Phoneme struct {
	Token    string
	Start    float64
	Onset    bool
	Language string
}

type placedNote struct {
	Note
	Start float64
	End   float64
}

type builtWord struct {
	start float64
	word  dsinfer.Word
}

// BuildWords converts note-level duration input into dsinfer words.
func BuildWords(
	speakers []string,
	mix [][]float64,
	mixSampleRate float64,
	notes []Note,
) ([]dsinfer.Word, error) {
	outputSpeakers, err := buildSpeakers(speakers)
	if err != nil {
		return nil, err
	}
	if err := validateMix(mix, mixSampleRate, len(outputSpeakers)); err != nil {
		return nil, err
	}
	placedNotes, err := placeNotes(notes)
	if err != nil {
		return nil, err
	}
	if len(placedNotes) == 0 {
		return []dsinfer.Word{}, nil
	}
	if isSlur(placedNotes[0]) {
		return nil, fmt.Errorf("dsinfer/builder: first note cannot be slur")
	}

	builtWords := make([]builtWord, 0, len(placedNotes))
	if placedNotes[0].Gap > 0 {
		header, _ := splitHeaderAndBody(phonemesOf(placedNotes[0]))
		phonemes := make([]dsinfer.Phoneme, 0, len(header)+1)
		phonemes = append(phonemes, newPhoneme(spPronunciation, placedNotes[0].Language, 0))
		for _, phoneme := range header {
			phonemes = append(phonemes, newPhoneme(phoneme.Token, phoneme.Language, placedNotes[0].Gap+phoneme.Start))
		}
		builtWords = append(builtWords, builtWord{
			start: 0,
			word: dsinfer.Word{
				Phonemes: phonemes,
				Notes: []dsinfer.Note{{
					Cent:     0,
					Duration: placedNotes[0].Gap,
					Rest:     true,
				}},
			},
		})
	}

	for noteIndex := 0; noteIndex < len(placedNotes); noteIndex++ {
		note := placedNotes[noteIndex]
		wordStart := note.Start
		wordEnd := note.End
		wordNotes := []dsinfer.Note{newNote(note)}
		_, body := splitHeaderAndBody(phonemesOf(note))
		wordPhonemes := make([]dsinfer.Phoneme, 0, len(body))
		for _, phoneme := range body {
			wordPhonemes = append(wordPhonemes, convertBodyPhoneme(phoneme))
		}

		for noteIndex+1 < len(placedNotes) && isSlur(placedNotes[noteIndex+1]) {
			next := placedNotes[noteIndex+1]
			wordNotes = append(wordNotes, newNote(next))
			wordEnd = next.End
			noteIndex++
		}

		if noteIndex+1 >= len(placedNotes) {
			if len(wordPhonemes) > 0 {
				builtWords = append(builtWords, builtWord{
					start: wordStart,
					word: dsinfer.Word{
						Phonemes: wordPhonemes,
						Notes:    wordNotes,
					},
				})
			}
			continue
		}

		next := placedNotes[noteIndex+1]
		gapLength := next.Start - wordEnd
		if gapLength < 0 {
			gapLength = 0
		}
		header, _ := splitHeaderAndBody(phonemesOf(next))
		nextHeader := make([]dsinfer.Phoneme, 0, len(header))
		attachBase := wordEnd - wordStart
		if gapLength > 0 {
			attachBase = gapLength
		}
		for _, phoneme := range header {
			nextHeader = append(nextHeader, newPhoneme(phoneme.Token, phoneme.Language, attachBase+phoneme.Start))
		}

		if gapLength == 0 {
			wordPhonemes = append(wordPhonemes, nextHeader...)
		}
		if len(wordPhonemes) > 0 {
			builtWords = append(builtWords, builtWord{
				start: wordStart,
				word: dsinfer.Word{
					Phonemes: wordPhonemes,
					Notes:    wordNotes,
				},
			})
		}
		if gapLength > 0 {
			gapPhonemes := make([]dsinfer.Phoneme, 0, len(nextHeader)+1)
			gapPhonemes = append(gapPhonemes, newPhoneme(spPronunciation, next.Language, 0))
			gapPhonemes = append(gapPhonemes, nextHeader...)
			builtWords = append(builtWords, builtWord{
				start: wordEnd,
				word: dsinfer.Word{
					Phonemes: gapPhonemes,
					Notes: []dsinfer.Note{{
						Cent:     placedNotes[noteIndex].Cents,
						Duration: gapLength,
						Rest:     true,
					}},
				},
			})
		}
	}

	result := make([]dsinfer.Word, 0, len(builtWords))
	for _, built := range builtWords {
		word := built.word
		for index := range word.Phonemes {
			sampleTime := built.start + word.Phonemes[index].Start
			proportions := sampleSpeakerProportion(mix, mixSampleRate, len(outputSpeakers), sampleTime)
			word.Phonemes[index].Speakers = buildPhonemeSpeakers(outputSpeakers, proportions)
		}
		result = append(result, word)
	}
	return result, nil
}

func buildSpeakers(speakers []string) ([]dsinfer.Speaker, error) {
	if len(speakers) == 0 {
		return nil, fmt.Errorf("dsinfer/builder: speakers cannot be empty")
	}
	result := make([]dsinfer.Speaker, 0, len(speakers))
	for _, speaker := range speakers {
		result = append(result, dsinfer.Speaker{ID: speaker})
	}
	return result, nil
}

func buildPhonemeSpeakers(speakers []dsinfer.Speaker, proportions []float64) []dsinfer.Speaker {
	result := make([]dsinfer.Speaker, len(speakers))
	for index, speaker := range speakers {
		result[index] = dsinfer.Speaker{
			ID:         speaker.ID,
			Proportion: proportions[index],
		}
	}
	return result
}

func validateMix(mix [][]float64, sampleRate float64, speakerCount int) error {
	if sampleRate <= 0 {
		return fmt.Errorf("dsinfer/builder: mix sample rate must be positive")
	}
	if speakerCount > 1 && len(mix) == 0 {
		return fmt.Errorf("dsinfer/builder: mix cannot be empty for multiple speakers")
	}
	expectedLength := speakerCount - 1
	for index, sample := range mix {
		if len(sample) != expectedLength {
			return fmt.Errorf("dsinfer/builder: mix sample %d has %d values, want %d", index, len(sample), expectedLength)
		}
	}
	return nil
}

func placeNotes(notes []Note) ([]placedNote, error) {
	result := make([]placedNote, 0, len(notes))
	var position float64
	for index, note := range notes {
		if note.Gap < 0 {
			return nil, fmt.Errorf("dsinfer/builder: note %d gap cannot be negative", index)
		}
		if note.Duration < 0 {
			return nil, fmt.Errorf("dsinfer/builder: note %d duration cannot be negative", index)
		}
		position += note.Gap
		placed := placedNote{
			Note:  note,
			Start: position,
			End:   position + note.Duration,
		}
		result = append(result, placed)
		position = placed.End
	}
	return result, nil
}

func splitHeaderAndBody(phonemes []Phoneme) ([]Phoneme, []Phoneme) {
	for index, phoneme := range phonemes {
		if phoneme.Onset {
			return phonemes[:index], phonemes[index:]
		}
	}
	return phonemes, nil
}

func phonemesOf(note placedNote) []Phoneme {
	if len(note.Phonemes) != 0 || !isRest(note) {
		return note.Phonemes
	}
	return []Phoneme{{
		Token:    note.Pronunciation,
		Start:    0,
		Onset:    true,
		Language: note.Language,
	}}
}

func newNote(note placedNote) dsinfer.Note {
	return dsinfer.Note{
		Cent:     note.Cents,
		Duration: note.Duration,
		Rest:     isRest(note),
	}
}

func convertBodyPhoneme(phoneme Phoneme) dsinfer.Phoneme {
	if phoneme.Token == spPronunciation || phoneme.Token == apPronunciation {
		return newPhoneme(phoneme.Token, phoneme.Language, 0)
	}
	return newPhoneme(phoneme.Token, phoneme.Language, phoneme.Start)
}

func newPhoneme(token string, language string, start float64) dsinfer.Phoneme {
	return dsinfer.Phoneme{
		Token:    token,
		Language: language,
		Start:    start,
	}
}

func sampleSpeakerProportion(mix [][]float64, sampleRate float64, speakerCount int, sampleTime float64) []float64 {
	if speakerCount == 1 {
		return []float64{1}
	}
	if sampleTime <= 0 || len(mix) == 1 {
		return completeSpeakerProportion(mix[0], speakerCount)
	}
	position := sampleTime * sampleRate
	lastIndex := len(mix) - 1
	if position >= float64(lastIndex) {
		return completeSpeakerProportion(mix[lastIndex], speakerCount)
	}
	leftIndex := int(math.Floor(position))
	rightIndex := leftIndex + 1
	weight := position - float64(leftIndex)
	sample := make([]float64, speakerCount-1)
	for index := range sample {
		sample[index] = mix[leftIndex][index]*(1-weight) + mix[rightIndex][index]*weight
	}
	return completeSpeakerProportion(sample, speakerCount)
}

func completeSpeakerProportion(sample []float64, speakerCount int) []float64 {
	result := make([]float64, speakerCount)
	var sum float64
	for index, value := range sample {
		result[index] = value
		sum += value
	}
	result[speakerCount-1] = 1 - sum
	return result
}

func isRest(note placedNote) bool {
	return note.Pronunciation == spPronunciation || note.Pronunciation == apPronunciation
}

func isSlur(note placedNote) bool {
	return note.Pronunciation == slurPronunciation
}
