package outcmds

type Cmd int16

const (
	GlobalCommands Cmd = iota
	SessionId
	Copy
	Kick
	ChatMessage
	NavigateTo
	SettingSet

	BlockUser
	UnblockUser
	FriendOnline
	FriendOffline
	FriendList
	FriendAdd
	FriendRemove
	InviteAdd
	InviteRemove

	InventoryOpen
	InventoryCoin
	InventoryUpdate

	ChatCommands
	Reload
	SetUser
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
	BoatTicks

	Bidding
	Playing
	Card
	Cards
	Sit
	PlayTo
	Take
	Scores
	Over
	OfferBlind

	LobbyStatus
)
