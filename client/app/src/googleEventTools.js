export const eventDuration = (event) => (parseFloat(((new Date(event.end.dateTime)) - (new Date(event.start.dateTime))) / (1000 * 3600)) || 0);
