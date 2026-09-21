console.log(sameAmount("abc123abc", /a/g, /1/g));
// Output: false (a appears twice, 1 appears once)

console.log(sameAmount("xoxo", /x/g, /o/g));
// Output: true (x appears twice, o appears twice)

console.log(sameAmount("hello world", /l/g, /o/g));
// Output: false (l appears 3 times, o appears 2 times)

console.log(sameAmount("no matches", /z/g, /x/g));
// Output: true (both appear 0 times)