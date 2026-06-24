package ai

type Message struct {
	Role    string
	Content string
}

type ContextManager struct {
	histroy []Message
	maxMsgs int
}

func NewContextManager(maxMsgs int) *ContextManager {
	return &ContextManager{
		histroy: make([]Message, 0),
		maxMsgs: maxMsgs,
	}
}

func (c *ContextManager) Add(Role, Content string) {
	c.histroy = append(c.histroy, Message{Role: Role, Content: Content})

	if len(c.histroy) > c.maxMsgs {
		c.histroy = c.histroy[len(c.histroy)-c.maxMsgs:]
	}
}

func (c *ContextManager) Get() []Message {
	return c.histroy
}
