package incmds

type Cmd int16

const (
	InventoryCmd Cmd = iota

	SettingSet
	SettingGetGroup
	LobbyListJoin
	EditorJoin
	BnavJoin
	LobbyJoin

	FriendInvite
	FriendDecline
	FriendAdd
	FriendRemove
	Block
	Unblock

	MapList
	CgMapList
	StructureSetList
	TileSetList

	StatsUser
	StatsTop
)

const LobbyCmds Cmd = 100

const (
	LobbyCreate Cmd = iota + LobbyCmds + 1
	LobbyApply
	ChatMessage

	BnavGetPositions
	BnavSavePosition

	Moves
	Shots
	Bomb
	Ready
	NextBoat
	SpawnSide
	Team
	WantMove
	Sync

	MapListAll
	MapGet
	MapSave
	MapCreate
	MapDelete
	TileSetGet
	StructureSetGet
	WeightSave

	Sit
	Kick
	Jump
	Bid
	Card
	DeclineBlind
)
