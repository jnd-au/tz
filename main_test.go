/**
 * This file is part of tz.
 *
 * tz is free software: you can redistribute it and/or modify it under
 * the terms of the GNU General Public License as published by the Free
 * Software Foundation, either version 3 of the License, or (at your
 * option) any later version.
 *
 * tz is distributed in the hope that it will be useful, but WITHOUT
 * ANY WARRANTY; without even the implied warranty of MERCHANTABILITY
 * or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU General Public
 * License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with tz.  If not, see <https://www.gnu.org/licenses/>.
 **/
package main

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"golang.org/x/tools/txtar"
)

var (
	ansiControlSequenceRegexp = regexp.MustCompile(regexp.QuoteMeta(termenv.CSI) + "[^m]*m")
	utcMinuteAfterMidnightTime = time.Date(
		2017, // Year
		11, // Month
		5, // Day
		0, // Hour
		1, // Minutes
		2, // Seconds
		127, // Nanoseconds
		time.UTC,
	)
	utcMinuteAfterMidnightModel = model{
		zones:      DefaultZones[len(DefaultZones) - 1:],
		clock:      *NewClockTime(utcMinuteAfterMidnightTime),
		isMilitary: true,
		showDates:  true,
	}
)

func getTimestampWithHour(hour int) time.Time {
	return time.Date(
		time.Now().Year(),
		time.Now().Month(),
		time.Now().Day(),
		hour,
		0, // Minutes set to 0
		0, // Seconds set to 0
		0, // Nanoseconds set to 0
		time.Now().Location(),
	)
}

func stripAnsiControlSequences(s string) string {
	return ansiControlSequenceRegexp.ReplaceAllString(s, "")
}

func stripAnsiControlSequencesAndNewline(bytes []byte) string {
	s := strings.TrimSuffix(string(bytes), "\n")
	return ansiControlSequenceRegexp.ReplaceAllString(s, "")
}

func TestUpdateIncHour(t *testing.T) {
	// "l" key -> go right
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'l'},
		Alt:   false,
	}

	tests := []struct {
		startHour          int
		nextHour           int
		changesClockDateBy int
	}{
		{startHour: 0, nextHour: 1, changesClockDateBy: 0},
		{startHour: 1, nextHour: 2, changesClockDateBy: 0},
		// ...
		{startHour: 23, nextHour: 0, changesClockDateBy: 1},
	}

	for _, test := range tests {
		m := model{
			zones: DefaultZones,
			clock: *NewClockTime(getTimestampWithHour(test.startHour)),
		}

		db := m.clock.Time().Day()
		_, cmd := m.Update(msg)
		da := m.clock.Time().Day()

		if cmd != nil {
			t.Errorf("Expected nil Cmd, but got %v", cmd)
			return
		}
		h := m.clock.t.Hour()
		if h != test.nextHour {
			t.Errorf("Expected %d, but got %d", test.nextHour, h)
		}
		if test.changesClockDateBy != 0 && da == db {
			t.Errorf("Expected date change of %d day, but got %d", test.changesClockDateBy, da-db)
		}
	}
}

func TestUpdateDecHour(t *testing.T) {
	// "h" key -> go left
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'h'},
		Alt:   false,
	}

	tests := []struct {
		startHour int
		nextHour  int
	}{
		{startHour: 23, nextHour: 22},
		{startHour: 22, nextHour: 21},
		// ...
		{startHour: 0, nextHour: 23},
	}

	for _, test := range tests {
		m := model{
			zones: DefaultZones,
			clock: *NewClockTime(getTimestampWithHour(test.startHour)),
		}
		_, cmd := m.Update(msg)
		if cmd != nil {
			t.Errorf("Expected nil Cmd, but got %v", cmd)
			return
		}
		h := m.clock.t.Hour()
		if h != test.nextHour {
			t.Errorf("Expected %d, but got %d", test.nextHour, h)
		}
	}
}

func TestUpdateQuitMsg(t *testing.T) {
	// "q" key -> quit
	msg := tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'q'},
		Alt:   false,
	}

	m := model{
		zones: DefaultZones,
		clock: *NewClockTime(getTimestampWithHour(10)),
	}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Errorf("Expected tea.Quit Cmd, but got %v", cmd)
		return
	}
	// tea.Quit is a function, we can't really test with == here, and
	// calling it is getting into internal territory.
}

func TestMilitaryTime(t *testing.T) {
	testDataFile := "testdata/main/test-military-time.txt"
	testData, err := txtar.ParseFile(testDataFile)
	if err != nil {
		t.Fatal(err)
	}

	formatted := utcMinuteAfterMidnightTime.Format(" 15:04, Mon Jan 02, 2006")
	expected := stripAnsiControlSequencesAndNewline(testData.Files[0].Data)
	observed := stripAnsiControlSequences(utcMinuteAfterMidnightModel.View())

	archive := txtar.Archive{
		Comment: testData.Comment,
		Files: []txtar.File{
			{
				Name: "expected",
				Data: []byte(expected),
			},
			{
				Name: "observed",
				Data: []byte(observed),
			},
		},
	}
	os.WriteFile(testDataFile, txtar.Format(&archive), 0666)

	if formatted != expected {
		t.Errorf("Expected military time of %s, but got %s", expected, formatted)
	}
	if !strings.Contains(observed, expected) {
		t.Errorf("Expected military time of %s, but got %s", expected, observed)
	}
}
