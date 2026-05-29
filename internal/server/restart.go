//ff:what 서버 재시작: Stop → Start 순서 실행
//ff:why Stop이 완전히 끝난 후에만 Start해야 포트 충돌 없음
package server

import (
	"context"

	"github.com/user/vpm/internal/config"
	"github.com/user/vpm/internal/logbuf"
)

// Restart stops then starts the server sequentially.
func (m *Manager) Restart(
	ctx context.Context,
	proj config.Project,
	srv config.Server,
	onState func(StateEvent),
	onCollision func(CollisionEvent),
	onLog logbuf.Flush,
) error {
	if err := m.Stop(ctx, proj, srv, onState); err != nil {
		return err
	}
	return m.Start(ctx, proj, srv, onState, onCollision, onLog)
}
