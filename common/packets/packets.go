package packets

type CreateUser struct {
	Name string `json:"name"`
}

type CreateUserAck struct {
	Name   string `json:"name"`
	UserID string `json:"id"`
}

type LoadGameState struct {
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
	UserID string `json:"id"`
	Name   string `json:"name"`
	Score  int64  `json:"highscore"`
}

type GetFriends struct {
	Friends []FriendData `json:"friends"`
}

type UserData struct {
	UserID string `json:"id"`
	Name   string `json:"name"`
}

type GetAllUsers struct {
	Users []UserData `json:"users"`
}
