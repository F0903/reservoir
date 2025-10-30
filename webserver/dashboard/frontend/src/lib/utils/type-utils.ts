// A helper type that recursively builds a union of numbers from 0 to N-1
type Enumerate<N extends number, Acc extends number[] = []> = Acc["length"] extends N
    ? Acc[number]
    : Enumerate<N, [...Acc, Acc["length"]]>;

// The main type that creates a range from F (from) to T (to)
// It works by getting all numbers from 0 to T and excluding numbers from 0 to F.
export type IntRange<F extends number, T extends number> = Exclude<Enumerate<T>, Enumerate<F>>;
