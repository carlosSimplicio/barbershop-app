const integers = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 0 });

export function formatOfferingPrice(cents: number): string {
  const value = BigInt(cents);
  return `R$\u00a0${integers.format(value / 100n)},${String(value % 100n).padStart(2, "0")}`;
}
