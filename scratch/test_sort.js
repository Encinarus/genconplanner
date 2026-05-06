const titles = [
    "Acquire",
    "Agricola",
    "12 Rivers",
    "4 Years to Mars",
    "A Carnivore did it",
    "A Feast for Odin",
    " Acquire", // Leading space
    "-Acquire", // Leading symbol
    "A.I.",
    "Unspecified"
];

const sortFn = (a, b) => a.localeCompare(b, 'en', { numeric: true, sensitivity: 'base' });

console.log("Sort Results (en, numeric, base):");
const sorted = [...titles].sort(sortFn);
sorted.forEach(t => console.log(`'${t}'`));
