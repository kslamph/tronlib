package account

const MaxRetries = 3

type Manager struct{}

func NewManager() *Manager { return &Manager{} }

func (m *Manager) Transfer() {}

func helper() {}
