package stream

// Diagnostic keeps command failures separate from log records, with bounded memory.
type Diagnostic struct{ data []byte }

func (d *Diagnostic) Write(data []byte) (int, error) {
	const limit = 32 * 1024
	n := len(data)
	if n >= limit {
		d.data = append(d.data[:0], data[n-limit:]...)
		return n, nil
	}
	if len(d.data)+n > limit {
		d.data = d.data[len(d.data)+n-limit:]
	}
	d.data = append(d.data, data...)
	return n, nil
}

func (d *Diagnostic) String() string { return string(d.data) }
