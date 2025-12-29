/**
 * Class name utility function that concatenates strings and arrays of strings recursively
 *
 * @param args - Any number of arguments that can be strings, arrays of strings, or nested arrays
 * @returns A single string with all class names concatenated and separated by spaces
 *
 * @example
 * cls('btn', 'primary') // 'btn primary'
 * cls(['btn', 'primary'], 'active') // 'btn primary active'
 * cls('btn', ['primary', ['active', 'focus']], 'large') // 'btn primary active focus large'
 * cls(['btn'], [['primary']], [[['active']]]) // 'btn primary active'
 */
export type ClsArg = string | readonly string[] | ClsArg[];

export function cls(...args: ClsArg[]): string {
  const classes: string[] = [];

  function processArg(arg: ClsArg): void {
    if (typeof arg === 'string') {
      if (arg.trim()) {
        classes.push(arg.trim());
      }
    } else if (Array.isArray(arg)) {
      arg.forEach(processArg);
    }
  }

  args.forEach(processArg);

  return classes.join(' ');
}
