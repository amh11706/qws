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
	SettingsGet
	StatsUser
	StatsTop

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

	MapList
	CgMapList
	StructureSetList
	TileSetList
)

const (
	LobbyCmds Cmd = iota + 100
	LobbyUpdate
	LobbyList
	LobbyRemove
	PlayerAdd
	PlayerList
	PlayerRemove

	BnavPositions
	BnavSavedPosition

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

	WeightSaved
	TileSet
	StructureSet
	Map
	Maps
	MapSaved
	MapCreated
	MapDeleted

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
