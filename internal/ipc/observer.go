/*
 * OpenFriend — Minecraft Java Edition Friends List bridge.
 * Copyright (c) 2026 ZSHARE (https://zpw.jp). Licensed under the MIT License.
 */
package ipc

import (
	"github.com/google/uuid"
	"jp.zpw.openfriend/internal/api"
	"jp.zpw.openfriend/internal/presence"
)

type writerObserver struct {
	writer *Writer
}

var _ presence.Observer = (*writerObserver)(nil)

func newWriterObserver(w *Writer) *writerObserver {
	return &writerObserver{writer: w}
}

func friendDtoToMap(d api.FriendDto) map[string]any {
	return map[string]any{
		"profileId": d.ProfileID.UUID().String(),
		"name":      d.Name,
	}
}

func (o *writerObserver) OnFriendAdded(d api.FriendDto) {
	_ = o.writer.Notify("friend.added", friendDtoToMap(d))
}

func (o *writerObserver) OnFriendRemoved(id uuid.UUID) {
	_ = o.writer.Notify("friend.removed", map[string]any{"profileId": id.String()})
}

func (o *writerObserver) OnIncomingRequest(d api.FriendDto) {
	_ = o.writer.Notify("friend.requestIncoming", friendDtoToMap(d))
}

func (o *writerObserver) OnIncomingResolved(id uuid.UUID) {
	_ = o.writer.Notify("friend.requestIncomingResolved", map[string]any{"profileId": id.String()})
}

func (o *writerObserver) OnOutgoingRequest(d api.FriendDto) {
	_ = o.writer.Notify("friend.requestOutgoing", friendDtoToMap(d))
}

func (o *writerObserver) OnOutgoingResolved(id uuid.UUID) {
	_ = o.writer.Notify("friend.requestOutgoingResolved", map[string]any{"profileId": id.String()})
}

func (o *writerObserver) OnInitialSnapshot(resp *api.FriendsListResponse) {
	rows := func(list []api.FriendDto) []map[string]any {
		out := make([]map[string]any, 0, len(list))
		for _, d := range list {
			out = append(out, friendDtoToMap(d))
		}
		return out
	}
	_ = o.writer.Notify("friends.snapshot", map[string]any{
		"friends":  rows(resp.Friends),
		"incoming": rows(resp.IncomingRequests),
		"outgoing": rows(resp.OutgoingRequests),
	})
}
