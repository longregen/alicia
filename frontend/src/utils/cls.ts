/**
 * Class name utility function that concatenates strings and arrays of strings recursively.
 * Compatible with conditional classes, booleans, null, and undefined values.
 *
 * @param args - Any number of arguments that can be strings, arrays, booleans, null, or undefined
 * @returns A single string with all class names concatenated and separated by spaces
 *
 * @example
 * cls('btn', 'primary') // 'btn primary'
 * cls(['btn', 'primary'], 'active') // 'btn primary active'
 * cls('btn', ['primary', ['active', 'focus']], 'large') // 'btn primary active focus large'
 * cls('btn', isActive && 'active', false && 'disabled') // 'btn active'
 * cls('btn', null, undefined, 'primary') // 'btn primary'
 */
export type ClsArg =
  | string
  | boolean
  | null
  | undefined
  | readonly ClsArg[]
  | ClsArg[];

export function cls(...args: ClsArg[]): string {
  const classes: string[] = [];

  function processArg(arg: ClsArg): void {
    if (!arg) {
      // Skip falsy values (false, null, undefined). Empty strings filtered below via trim().
      return;
    }

    if (typeof arg === 'string') {
      if (arg.trim()) {
        classes.push(arg.trim());
      }
    } else if (Array.isArray(arg)) {
      arg.forEach(processArg);
    }
    // Booleans that are true are skipped (they don't add classes)
  }

  args.forEach(processArg);

  return classes.join(' ');
}
