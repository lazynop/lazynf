package engine

type ConflictKind int

const (
	ConflictAlreadyImported ConflictKind = iota
	ConflictFilesOnDisk
	ConflictVersionDowngrade
)

type ConflictChoice int

const (
	ChoiceCancel ConflictChoice = iota
	ChoiceSkip
	ChoiceForce
	ChoiceImportAs
)
