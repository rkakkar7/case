package packets

type CreateUser struct {
	Error int    `json:"error,omitempty"`
	Name  string `json:"name"`
}

type CreateUserAck struct {
	Error  int    `json:"error,omitempty"`
	Name   string `json:"name"`
	UserID string `json:"id"`
}

type LoadGameState struct {
	Error       int   `json:"error,omitempty"`
	GamesPlayed int64 `json:"gamesPlayed"`
	Score       int64 `json:"score"`
}

type SaveGameState struct {
	GamesPlayed int64 `json:"gamesPlayed"`
	Score       int64 `json:"score"`
}

type UpdateFriends struct {
	Friends []string `json:"friends"`
}

type FriendData struct {
	Error  int    `json:"error,omitempty"`
	UserID string `json:"id"`
	Name   string `json:"name"`
	Score  int64  `json:"highscore"`
}

type GetFriends struct {
	Error   int          `json:"error,omitempty"`
	Friends []FriendData `json:"friends"`
}

type UserData struct {
	UserID string `json:"id"`
	Name   string `json:"name"`
}

type GetAllUsers struct {
	Error int        `json:"error,omitempty"`
	Users []UserData `json:"users"`
}
