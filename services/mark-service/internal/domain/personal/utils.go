package personal

type Changeable interface {
	GetID() uint
	GetRevision() uint
	IsDeleted() bool
}

func toChanges[T Changeable](objs []T, upTo uint, limit int) Changes[T] {
	out := Changes[T]{Upserted: []T{}, Removed: []uint{}, Cursor: upTo}
	if len(objs) > limit {
		objs = objs[:limit]
		out.HasMore = true
		out.Cursor = objs[len(objs)-1].GetRevision()
	}
	for _, m := range objs {
		if m.IsDeleted() {
			out.Removed = append(out.Removed, m.GetID())
		} else {
			out.Upserted = append(out.Upserted, m)
		}
	}
	return out
}
