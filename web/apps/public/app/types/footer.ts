/** A footer link: internal route or external URL. */
export type FooterLink = {
  label: string;
  to: string;
  external?: boolean;
};
