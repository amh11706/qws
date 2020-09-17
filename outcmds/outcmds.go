package outcmds

type Cmd int16

const (
	SessionId Cmd = iota
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

const LobbyCmds Cmd = 100

const (
	LobbyUpdate Cmd = iota + LobbyCmds + 1
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
