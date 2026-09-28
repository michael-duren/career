ALTER TABLE goals
ADD COLUMN selected_weekdays JSONB NOT NULL DEFAULT '[1, 2, 3, 4, 5, 6, 7]'::jsonb,
ADD CONSTRAINT goals_selected_weekdays_array CHECK (
    jsonb_typeof(selected_weekdays) = 'array'
    AND selected_weekdays <@ '[1, 2, 3, 4, 5, 6, 7]'::jsonb
);
