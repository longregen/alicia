// Quotes from the heroes of Apple's "Think Different" campaign (1997)
// "Here's to the crazy ones. The misfits. The rebels. The troublemakers."

export interface Quote {
  text: string;
  author: string;
}

export const thinkDifferentQuotes: Quote[] = [
  // Albert Einstein (verified: Saturday Evening Post, 1929)
  { text: "Imagination is more important than knowledge. Knowledge is limited. Imagination encircles the world.", author: "Albert Einstein" },

  // Bob Dylan (verified: "It's Alright, Ma (I'm Only Bleeding)", 1965)
  { text: "He not busy being born is busy dying.", author: "Bob Dylan" },
  { text: "All I can do is be me, whoever that is.", author: "Bob Dylan" },
  { text: "An artist has got to be careful never really to arrive at a place where he thinks he's AT somewhere. You always have to realize that you're constantly in a state of becoming.", author: "Bob Dylan" },

  // Martin Luther King Jr. (verified: Letter from Birmingham Jail & Strength to Love, 1963)
  { text: "Injustice anywhere is a threat to justice everywhere.", author: "Martin Luther King Jr." },
  { text: "The time is always right to do what is right.", author: "Martin Luther King Jr." },
  { text: "Darkness cannot drive out darkness; only light can do that. Hate cannot drive out hate; only love can do that.", author: "Martin Luther King Jr." },

  // John Lennon
  { text: "Reality leaves a lot to the imagination.", author: "John Lennon" },
  { text: "If everyone demanded peace instead of another television set, then there'd be peace.", author: "John Lennon" },

  // Buckminster Fuller
  { text: "You cannot change how someone thinks, but you can give them a tool to use which will lead them to think differently.", author: "Buckminster Fuller" },
  { text: "We are called to be architects of the future, not its victims.", author: "Buckminster Fuller" },

  // Thomas Edison (verified: Harper's Monthly, 1932)
  { text: "Genius is one percent inspiration and ninety-nine percent perspiration.", author: "Thomas Edison" },
  { text: "Many of life's failures are people who did not realize how close they were to success when they gave up.", author: "Thomas Edison" },

  // Muhammad Ali
  { text: "He who is not courageous enough to take risks will accomplish nothing in life.", author: "Muhammad Ali" },
  { text: "I am the greatest, I said that even before I knew I was.", author: "Muhammad Ali" },

  // Mahatma Gandhi (verified: All Men Are Brothers)
  { text: "The weak can never forgive. Forgiveness is the attribute of the strong.", author: "Mahatma Gandhi" },
  { text: "Truth never damages a cause that is just.", author: "Mahatma Gandhi" },

  // Amelia Earhart
  { text: "Courage is the price that life exacts for granting peace.", author: "Amelia Earhart" },
  { text: "Women, like men, should try to do the impossible. And when they fail, their failure should be a challenge to others.", author: "Amelia Earhart" },

  // Alfred Hitchcock
  { text: "There is something more important than logic: imagination.", author: "Alfred Hitchcock" },
  { text: "I'm frightened of nothing except losing my creativity.", author: "Alfred Hitchcock" },
  { text: "It is how you do it, and not your content that makes you an artist.", author: "Alfred Hitchcock" },

  // Martha Graham
  { text: "There is a vitality, a life force, an energy, a quickening that is translated through you into action, and because there is only one of you in all of time, this expression is unique.", author: "Martha Graham" },
  { text: "Dance is the hidden language of the soul of the body.", author: "Martha Graham" },

  // Jim Henson
  { text: "I've got a dream too, but it's about singing and dancing and making people happy. That's the kind of dream that gets better the more people you share it with.", author: "Jim Henson" },
  { text: "When I was young, my ambition was to be one of the people who made a difference in this world. My hope is to leave the world a little better for having been there.", author: "Jim Henson" },

  // Frank Lloyd Wright
  { text: "The longer I live the more beautiful life becomes.", author: "Frank Lloyd Wright" },
  { text: "Study nature, love nature, stay close to nature. It will never fail you.", author: "Frank Lloyd Wright" },

  // Pablo Picasso
  { text: "Art is a lie that makes us realize truth.", author: "Pablo Picasso" },

  // Maria Callas
  { text: "I will always be as difficult as necessary to achieve the best.", author: "Maria Callas" },
  { text: "An opera begins long before the curtain goes up and ends long after it has come down. It starts in my imagination, it becomes my life, and it stays part of my life long after I've left the opera house.", author: "Maria Callas" },
  { text: "You are born an artist or you are not. And you stay an artist, dear, even if your voice is less of a fireworks. The artist is always there.", author: "Maria Callas" },

  // Ted Turner
  { text: "You should set goals beyond your reach so you always have something to live for.", author: "Ted Turner" },
  { text: "To succeed you have to be innovative.", author: "Ted Turner" },
  { text: "I'd rather be known for making mistakes than for doing nothing.", author: "Ted Turner" },

  // Richard Branson
  { text: "You don't learn to walk by following rules. You learn by doing, and by falling over.", author: "Richard Branson" },
  { text: "If people aren't calling you crazy, you aren't thinking big enough.", author: "Richard Branson" },
];

export function getRandomQuote(): Quote {
  const index = Math.floor(Math.random() * thinkDifferentQuotes.length);
  return thinkDifferentQuotes[index];
}
