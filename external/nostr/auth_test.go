package nostr

import (
	"errors"
	"testing"
)

func TestParsingHeaderSuccessful(t *testing.T) {
	header := "Nostr eyJpZCI6IjhlY2JkY2RkNTMyOTIwMDEwNTUyNGExNDI4NzkxMzg4MWIzOWQxNDA5ZDhiOTBjY2RiNGI0M2Y4ZjBmYzlkMGMiLCJwdWJrZXkiOiI5ZjBjYzE3MDIzYjJjZjUwOWUwZjFkMzA1NzkzZDIwZTdjNzIyNzY5MjhmZDliZjg1NTM2ODg3YWM1NzBhMjgwIiwiY3JlYXRlZF9hdCI6MTcwODc3MTIyNywia2luZCI6MjQyNDIsInRhZ3MiOltbInQiLCJnZXQiXSxbImV4cGlyYXRpb24iLCIxNzA4ODU3NTQwIl1dLCJjb250ZW50IjoiR2V0IEJsb2JzIiwic2lnIjoiMDJmMGQyYWIyM2IwNDQ0NjI4NGIwNzFhOTVjOThjNjE2YjVlOGM3NWFmMDY2N2Y5NmNlMmIzMWM1M2UwN2I0MjFmOGVmYWRhYzZkOTBiYTc1NTFlMzA4NWJhN2M0ZjU2NzRmZWJkMTVlYjQ4NTFjZTM5MGI4MzI4MjJiNDcwZDIifQ=="

	event, err := ParseNostrHeader(header)
	if err != nil {
		t.Errorf("parseNostrHeader(header) %+v", err)
	}

	err = ValidateAuthEvent(event)
	if err != nil {
        if !errors.Is(err,ErrEventExpired) {
		    t.Errorf("Validation failed. %+v", err)
        }

	}

}
