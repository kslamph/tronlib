package account

const MaxRetries = 3

type Manager struct{}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) Transfer() {}

func helper() {}

// internal is unexported: its exported-looking methods are NOT part of the
// public surface and must not appear in the migration guide.
type internal struct{}

func (i *internal) InternalOnly() {}
