package incmds

type Cmd int16

const (
	GlobalCommands Cmd = iota
	InventoryCmd
	ChatCommand

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
	RanksUser
	RanksTop
	MatchesUser
	MatchData
	MatchAi

	ChangeEmail
	ChangePass
	ChangeName
	SearchNames
	SearchNamesOnline
)

const (
	LobbyCmds Cmd = iota + 100
	LobbyCreate
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
	TMapSetGet

	Sit
	Kick
	Jump
	Bid
	Card
	DeclineBlind

	ShuffleTeams
)
