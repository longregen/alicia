import * as React from 'react'

import { cls } from '../../utils/cls'

function Input({
  className,
  ...props
}: React.ComponentProps<'input'>) {
  return (
    <input
      data-slot="input"
      className={cls(
        'w-full px-3 py-2 bg-background border border-border rounded-md text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-accent disabled:cursor-not-allowed disabled:opacity-50',
        className,
      )}
      {...props}
    />
  )
}

export { Input }
