import React from 'react';
import Button, { type ButtonProps } from './Button';

/**
 * PrimaryButton atom component for primary actions.
 * Pre-configured Button with primary variant.
 */

export interface PrimaryButtonProps extends Omit<ButtonProps, 'variant'> {
  // Inherits all ButtonProps except variant (always primary)
}

const PrimaryButton: React.FC<PrimaryButtonProps> = (props) => {
  return <Button {...props} variant="primary" />;
};

export default PrimaryButton;
