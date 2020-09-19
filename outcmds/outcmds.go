package outcmds

type Cmd int16

const (
	GlobalCommands Cmd = iota
	SessionId
	Copy
	Kick
	ChatMessage
	NavigateTo
	LobbyInvite
	SettingSet

	BlockUser
	UnblockUser
	FriendOnline
	FriendOffline
	FriendList
	FriendInvite
	FriendAdd
	FriendRemove
	InviteAdd
	InviteRemove

	IntentoryOpen
	InventoryCoin
	InventoryUpdate
)

const (
	LobbyCmds Cmd = iota + 100
	LobbyUpdate
	LobbyList
	LobbyRemove
	PlayerAdd
	PlayerList
	PlayerRemove

	LobbyJoin
	Sync
	NewBoat
	DelBoat
	Team
	Turn
	Moves
	Bomb
	Ready
	BoatTick

	Bidding
	Playing
	Card
	Cards
	Sit
	PlayTo
	Take
	Score
	Scores
	Over
	OfferBlind
)
