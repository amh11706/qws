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
	RanksTop3
	MatchesUser
	GetWinLoss
	MatchData
	MatchAi
	MatchScore
	MatchTraining

	ChangeEmail
	ChangePass
	ChangeName
	SearchNames
	SearchNamesOnline
	SetUserEmoji
	WatchQueue
	UnwatchQueue
	JoinQueue
	LeaveQueue
	RateMap
	GetBotMatch
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
	WantManeuver
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
	SetMapData
	SetMyJobbers
	Vote

	BASettings
	BASettingsGet
	BADamageReport
	BAToggleSink
	BAAddBoat
	BARemoveBoat
)
