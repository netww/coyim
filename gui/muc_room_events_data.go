package gui

import "github.com/coyim/coyim/session/muc"

type occupantSelfJoinedEvent struct {
	nickname string
}

type occupantLeftEvent struct {
	nickname string
}

type occupantJoinedEvent struct {
	nickname string
}

type occupantUpdatedEvent struct {
	nickname string
}

type nicknameConflictEvent struct {
	nickname string
}

type registrationRequiredEvent struct {
	nickname string
}

type roomInfoReceivedEvent struct {
	info *muc.RoomListing
}

type roomInfoTimeoutEvent struct{}

type loggingEnabledEvent struct{}

type loggingDisabledEvent struct{}

type messageEvent struct {
	tp       string
	nickname string
	message  string
}

type subjectEvent struct {
	subject string
}

type messageForbidden struct{}

type messageNotAcceptable struct{}

type roomViewEvent interface {
	markAsRoomViewEvent()
}

func (occupantLeftEvent) markAsRoomViewEvent()         {}
func (occupantJoinedEvent) markAsRoomViewEvent()       {}
func (occupantUpdatedEvent) markAsRoomViewEvent()      {}
func (occupantSelfJoinedEvent) markAsRoomViewEvent()   {}
func (messageEvent) markAsRoomViewEvent()              {}
func (subjectEvent) markAsRoomViewEvent()              {}
func (nicknameConflictEvent) markAsRoomViewEvent()     {}
func (registrationRequiredEvent) markAsRoomViewEvent() {}
func (roomInfoReceivedEvent) markAsRoomViewEvent()     {}
func (roomInfoTimeoutEvent) markAsRoomViewEvent()      {}
func (loggingEnabledEvent) markAsRoomViewEvent()       {}
func (loggingDisabledEvent) markAsRoomViewEvent()      {}
func (messageForbidden) markAsRoomViewEvent()          {}
func (messageNotAcceptable) markAsRoomViewEvent()      {}
