import { SelectField, type SelectFieldProps } from "@/components/ui";
import { BRAZILIAN_TIME_ZONES } from "@/lib/establishment/brazilian-time-zones";

type BrazilianTimeZoneFieldProps = Omit<SelectFieldProps, "children"> & {
  emptyOptionLabel?: string;
};

export function BrazilianTimeZoneField({
  emptyOptionLabel,
  ...props
}: BrazilianTimeZoneFieldProps) {
  return (
    <SelectField {...props}>
      {emptyOptionLabel ? (
        <option disabled value="">
          {emptyOptionLabel}
        </option>
      ) : null}
      {BRAZILIAN_TIME_ZONES.map((timeZone) => (
        <option key={timeZone.value} value={timeZone.value}>
          {timeZone.label}
        </option>
      ))}
    </SelectField>
  );
}
