package queue

type Track struct {
	Platform string
	ID       string
	URL      string
}

type Queue struct {
	tracks []Track
	index  int
}

func New() *Queue {
	return &Queue{
		tracks: make([]Track, 0),
		index:  -1,
	}
}

func (q *Queue) Add(track Track) {
	q.tracks = append(q.tracks, track)
}

func (q *Queue) Next() *Track {
	if q.index+1 < len(q.tracks) {
		q.index++
		return &q.tracks[q.index]
	}
	return nil
}

func (q *Queue) Current() *Track {
	if q.index >= 0 && q.index < len(q.tracks) {
		return &q.tracks[q.index]
	}
	return nil
}

func (q *Queue) Size() int {
	return len(q.tracks)
}
