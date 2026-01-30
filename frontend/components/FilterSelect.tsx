'use client';

interface Option {
  value: string;
  label: string;
}

interface FilterSelectProps {
  label: string;
  options: Option[];
  value: string;
  onChange: (value: string) => void;
}

export default function FilterSelect({ label, options, value, onChange }: FilterSelectProps) {
  return (
    <div className="flex flex-col gap-1">
      <label className="text-sm text-dark-400">{label}</label>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="bg-dark-800 border border-dark-700 text-white rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent cursor-pointer hover:bg-dark-700 transition-colors"
      >
        {options.map((option) => (
          <option key={option.value} value={option.value}>
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}
