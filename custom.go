package form

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/df-mc/dragonfly/server/player/form"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/google/uuid"
	"github.com/samber/lo"
)

// Custom represents a form that may be sent to a player and has fields that should be filled out by the player that the
// form is sent to.
type Custom struct {
	// Title is the title of the form that is displayed at the very top of the form.
	Title string
	// Buttons is a slice of elements that can be modified by a player. There must be at least one element for the client
	// to render the form.
	Elements []IdentifiedElement
	// Submit is called when the player pressed the submit button. This is always called after the Submit of every Element.
	// The values will be passed in a slice, with the same order as the Elements slice. If the form was closed, the values
	// slice will be nil.
	Submit func(submitter form.Submitter, tx *world.Tx, values map[uuid.UUID]any)
	// Close is called when the player closes a form.
	Close func(submitter form.Submitter, tx *world.Tx)
}

// Element appends an element to the bottom of the form.
func (form *Custom) Element(element IdentifiedElement) {
	form.Elements = append(form.Elements, element)
}

// SubmitJSON ...
func (form *Custom) SubmitJSON(data []byte, submitter form.Submitter, tx *world.Tx) error {
	if data == nil {
		if form.Close != nil {
			form.Close(submitter, tx)
		}
		return nil
	}
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.UseNumber()
	var inputData []any
	if err := dec.Decode(&inputData); err != nil {
		return fmt.Errorf("error decoding JSON data to slice: %w", err)
	}
	dataMap := make(map[uuid.UUID]any)
	elements := form.Elements
	if len(elements) != len(inputData) {
		elements = make([]IdentifiedElement, 0)
		for _, element := range form.Elements {
			switch element.Value().(type) {
			case Divider, Header, Label:
			default:
				elements = append(elements, element)
			}
		}
		if len(elements) != len(inputData) {
			return fmt.Errorf("form JSON data array does not have enough values")
		}
	}
	for i, element := range elements {
		err := element.Value().submit(inputData[i])
		if err != nil {
			return fmt.Errorf("error parsing form response value: %w", err)
		}
		dataMap[element.Key()] = inputData[i]
	}
	if form.Submit != nil {
		form.Submit(submitter, tx, dataMap)
	}
	return nil
}

// MarshalJSON ...
func (form *Custom) MarshalJSON() ([]byte, error) {
	if len(form.Elements) == 0 {
		return nil, errors.New("menu form requires at least one element")
	}
	return json.Marshal(map[string]any{
		"type":  "custom_form",
		"title": form.Title,
		"content": lo.Map(form.Elements, func(item IdentifiedElement, index int) Element {
			return item.Value()
		}),
	})
}
