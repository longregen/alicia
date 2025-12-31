import { Loader2Icon } from 'lucide-react'

import { cls } from '../../utils/cls'

function Spinner({ className, ...props }: React.ComponentProps<'svg'>) {
  return (
    <Loader2Icon
      role="status"
      aria-label="Loading"
      className={cls('size-4 animate-spin', className)}
      {...props}
    />
  )
}

export { Spinner }
