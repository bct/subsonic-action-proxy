package main

import (
	"reflect"
	"testing"
)

func TestCommandParsing(t *testing.T) {
  testCases := []string {
    "/bin/onkyo-ri-send-command 0 26 0xd9 0x20",
    "/bin/sh -c 'echo success; echo done'",
  }

	var parsedCommands []command
  for i := range testCases {
    command := command{}
    command.Set(testCases[i])
    parsedCommands = append(parsedCommands, command)
  }

	expected := [][]string{
		{"/bin/onkyo-ri-send-command", "0", "26", "0xd9", "0x20"},
		{"/bin/sh", "-c", "echo success; echo done"},
	}

	for i := range expected {
		// cast or else it won't compare successfully (and renders confusingly)
		actual := []string(parsedCommands[i])
		if !reflect.DeepEqual(expected[i], actual) {
			t.Fatalf("expected %+q, got %+q", expected[i], actual)
		}
	}
}
