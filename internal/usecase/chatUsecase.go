package usecase

// chatUsecase owns chat, message, membership, pin and permission logic.
// Methods land in a later pass; this is the scaffold constructor only.
type chatUsecase struct{}

func NewChatUsecase() *chatUsecase { return &chatUsecase{} }
